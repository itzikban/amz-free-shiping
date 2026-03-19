"use client";

import { Fragment, useMemo, useState } from "react";
import type { CheckResponse, FillToThresholdResponse } from "@/lib/types";
import { useI18n } from "@/lib/i18n";

type FormState = {
  url: string;
  country: "US" | "IL" | "NL";
  zip: string;
};

const SAMPLE = "https://www.amazon.com/dp/B0DHCZBKW7";

/**
 * Interactive homepage that lets users analyze product URLs, view analysis results, and add items to a watchlist.
 *
 * Provides a URL submission form with destination and optional US ZIP, a fetch-method toggle, result and error display,
 * a button to add the analyzed item to the user's watchlist, and a loading state that shows placeholder recommendations.
 *
 * @returns The rendered React element for the product analysis home page.
 */
export default function HomePage() {
  const { t } = useI18n();
  const [form, setForm] = useState<FormState>({ url: SAMPLE, country: "IL", zip: "" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<CheckResponse | null>(null);
  const [fillToThresholdResult, setFillToThresholdResult] = useState<FillToThresholdResponse | null>(null);
  const [submittedForm, setSubmittedForm] = useState<FormState | null>(null);
  const [addingWatchlist, setAddingWatchlist] = useState(false);
  const [watchlistAdded, setWatchlistAdded] = useState(false);
  const [fetchMethod, setFetchMethod] = useState<"auto" | "http">("auto");

  const canSubmit = useMemo(() => {
    if (!form.url.startsWith("http")) return false;
    // Only US requires ZIP code
    if (form.country === "US" && !form.zip.trim()) return false;
    return true;
  }, [form]);

  const isValidHttpUrl = (value: string): boolean => {
    try {
      const parsed = new URL(value);
      return parsed.protocol === "http:" || parsed.protocol === "https:";
    } catch {
      return false;
    }
  };

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit) return;

    const submitted: FormState = {
      url: form.url.trim(),
      country: form.country,
      zip: form.zip.trim(),
    };

    setLoading(true);
    setError(null);
    setResult(null);
    setFillToThresholdResult(null);
    setSubmittedForm(null);
    setWatchlistAdded(false);

    try {
      const params = new URLSearchParams({ url: submitted.url, country: submitted.country });
      if (submitted.country === "US" && submitted.zip) params.set("zip", submitted.zip);
      if (fetchMethod === "http") params.set("method", "http");
      const res = await fetch(`/api/check?${params.toString()}`);
      const body = await res.json();
      if (!res.ok) throw new Error(body?.error || "Request failed");
      setResult(body);

      const fillRes = await fetch(`/api/fill-to-threshold?${params.toString()}&threshold=50`);
      if (fillRes.ok) {
        const fillBody = await fillRes.json();
        setFillToThresholdResult(fillBody);
      }

      setSubmittedForm(submitted);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }

  async function addToWatchlist() {
    if (!submittedForm || addingWatchlist) return;
    setError(null);
    setAddingWatchlist(true);
    setWatchlistAdded(false);
    try {
      const res = await fetch("/api/v1/me/tracked-items", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          url: submittedForm.url,
          country: submittedForm.country,
          zip: submittedForm.country === "US" ? submittedForm.zip : "",
        }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body?.error || t("err_add_watchlist"));
      }
      setWatchlistAdded(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("err_network"));
      setWatchlistAdded(false);
    } finally {
      setAddingWatchlist(false);
    }
  }

  const mockRecommendations = [
    { id: 1, titleKey: "rec_title_1", price: "$345", tagKey: "rec_tag_free_shipping", cls: "img-placeholder-1" },
    { id: 2, titleKey: "rec_title_2", price: "$348", tagKey: "rec_tag_best_match", cls: "img-placeholder-2" },
    { id: 3, titleKey: "rec_title_3", price: "$429", tagKey: "rec_tag_free_shipping", cls: "img-placeholder-3" },
  ] as const;

  type MockRecommendation = (typeof mockRecommendations)[number];

  type RecommendationCard = {
    id: number;
    type: "mock" | "real";
    price: string;
  } & (
    | { type: "mock"; titleKey: MockRecommendation["titleKey"]; tagKey: MockRecommendation["tagKey"]; cls: string }
    | { type: "real"; title: string; imageUrl?: string; url?: string; freeShipping: boolean }
  );

  const recommendations: RecommendationCard[] = result?.alternatives && result.alternatives.length > 0
    ? result.alternatives.map((alt, i) => ({
        id: i + 1,
        type: "real" as const,
        title: alt.title,
        price: alt.price_usd != null ? `$${alt.price_usd.toFixed(2)}` : "N/A",
        imageUrl: alt.image_url,
        url: alt.url,
        freeShipping: alt.free_shipping,
      }))
    : mockRecommendations.map((r) => ({
        id: r.id,
        type: "mock" as const,
        titleKey: r.titleKey,
        tagKey: r.tagKey,
        price: r.price,
        cls: r.cls,
      }));

  const thresholdUsd = 50;
  const currentPrice = fillToThresholdResult?.current_price ?? result?.price_usd ?? null;
  const missingToThreshold = fillToThresholdResult?.missing_amount
    ?? (currentPrice != null ? Math.max(0, thresholdUsd - currentPrice) : 0);
  const showFillToThreshold = currentPrice != null && currentPrice < thresholdUsd && result?.free_shipping_country === false;

  const rankedFillCandidates = useMemo(() => {
    if (!showFillToThreshold) return [];

    if (fillToThresholdResult?.suggestions?.length) {
      return fillToThresholdResult.suggestions.slice(0, 2).map((s) => ({
        id: s.asin,
        title: s.title,
        price: s.price_usd,
        score: s.score,
        freeShipping: s.free_shipping_hint,
        url: s.url,
        imageUrl: s.image_url,
      }));
    }

    if (!result?.alternatives?.length) return [];

    return result.alternatives
      .filter((alt) => alt.price_usd != null && alt.price_usd > 0)
      .map((alt) => {
        const price = alt.price_usd as number;
        const overspend = Math.max(0, price - missingToThreshold);
        const gap = Math.abs(price - missingToThreshold);
        const shippingBonus = alt.free_shipping ? -2 : 0;
        const score = gap + overspend * 2 + shippingBonus;
        return {
          id: alt.asin,
          title: alt.title,
          price,
          score,
          freeShipping: alt.free_shipping,
          url: alt.url,
          imageUrl: alt.image_url,
        };
      })
      .sort((a, b) => a.score - b.score)
      .slice(0, 2);
  }, [fillToThresholdResult?.suggestions, missingToThreshold, result?.alternatives, showFillToThreshold]);

  const fallbackFillToThresholdSuggestions = [
    { id: "ft1", title: t("fill_to_50_item_1"), price: 5.99, url: undefined, imageUrl: undefined },
    { id: "ft2", title: t("fill_to_50_item_2"), price: 9.99, url: undefined, imageUrl: undefined },
    { id: "ft3", title: t("fill_to_50_item_3"), price: 14.99, url: undefined, imageUrl: undefined },
  ].sort((a, b) => Math.abs(a.price - missingToThreshold) - Math.abs(b.price - missingToThreshold));

  const fillToThresholdSuggestions = rankedFillCandidates.length > 0
    ? rankedFillCandidates
    : fallbackFillToThresholdSuggestions.slice(0, 2).map((fallback, idx) => {
        const alt = result?.alternatives?.[idx];
        if (!alt) return fallback;
        return {
          ...fallback,
          imageUrl: alt.image_url || fallback.imageUrl,
          url: alt.url || fallback.url,
          freeShipping: alt.free_shipping,
        };
      });

  const bestFillCombo = fillToThresholdResult?.combos?.[0] ?? null;

  return (
    <main className="container nexusHome">
      <section className="heroCenter animate-flow-1">
        <h1>{t("home_discover_title")}</h1>
        <p>{t("app_subtitle")}</p>

        <form className="search-container" onSubmit={onSubmit}>
          <div className="search-row grow">
            <label className="sr-only" htmlFor="product-url">{t("product_url")}</label>
            <input
              id="product-url"
              className="clean-input"
              type="url"
              value={form.url}
              onChange={(e) => setForm((x) => ({ ...x, url: e.target.value }))}
              placeholder={t("home_url_placeholder")}
              required
            />
          </div>

          <div className="search-row fixed">
            <label className="sr-only" htmlFor="destination-country">{t("destination_country")}</label>
            <select
              id="destination-country"
              className="clean-input"
              value={form.country}
              onChange={(e) => setForm((x) => ({ ...x, country: e.target.value as "US" | "IL" | "NL" }))}
            >
              <option value="IL">{t("home_country_il")}</option>
              <option value="US">{t("home_country_us")}</option>
              <option value="NL">{t("home_country_nl")}</option>
            </select>
          </div>

          {form.country === "US" && (
            <div className="search-row fixed">
              <label className="sr-only" htmlFor="destination-zip">{t("zip_us")}</label>
              <input
                id="destination-zip"
                className="clean-input"
                type="text"
                value={form.zip}
                onChange={(e) => setForm((x) => ({ ...x, zip: e.target.value }))}
                placeholder={t("home_zip_placeholder")}
              />
            </div>
          )}

          <button className="analyzeBtn" disabled={!canSubmit || loading}>
            {loading ? t("loading") : t("home_analyze")}
          </button>
        </form>

      </section>

      {(result || error) && (
        <section className="fluid-card targetCard animate-flow-2">
          {result?.image_url ? (
            <img className="targetImage" src={result.image_url} alt={result.title || "Product"} />
          ) : (
            <div className="targetImage img-placeholder-main" />
          )}
          <div className="targetBody">
            <h3>{result?.title || t("home_target_analysis")}</h3>
            {error && <p className="error">{error}</p>}
            {result && (
              <>
                <p className="mutedText">{result.url}</p>
                <div className="targetMeta">
                  <span className="priceTag">{result.price_usd != null ? `$${result.price_usd.toFixed(2)}` : "-"}</span>
                  <span className={`signalPill ${result.free_shipping_country ? "ok" : "no"}`}>
                    {result.free_shipping_country ? t("free_for_destination") : t("not_free_for_destination")}
                  </span>
                </div>
                <div className="chipRow">
                  <span className="signalPill neutral">{result.country}</span>
                  <span className="signalPill neutral" title={result.signal}>{result.signal.length > 30 ? result.signal.slice(0, 30) + "…" : result.signal}</span>
                </div>

                {showFillToThreshold && (
                  <div style={{ marginTop: 12 }}>
                    <div className="chipRow" style={{ marginBottom: 8 }}>
                      <span className="signalPill neutral">{t("fill_to_50_badge")}</span>
                      <span className="signalPill ok">{`${t("fill_to_50_missing_prefix")} $${missingToThreshold.toFixed(2)}`}</span>
                    </div>
                    <p className="mutedText" style={{ marginTop: 0 }}>{t("fill_to_50_subtitle")}</p>
                    <div className="recommendGrid" style={{ marginTop: 8 }}>
                      {fillToThresholdSuggestions.map((s) => {
                        const canOpen = !!(s.url && isValidHttpUrl(s.url));
                        const title = `${s.title} · $${s.price.toFixed(2)}`;
                        const card = (
                          <>
                            {s.imageUrl ? (
                              <img src={s.imageUrl} alt={s.title} className="recImage" style={{ objectFit: "cover" }} />
                            ) : (
                              <div className="recImage img-placeholder-2" />
                            )}
                            <h4>{s.title}</h4>
                            <div className="recMeta">
                              <strong>{`$${s.price.toFixed(2)}`}</strong>
                              <div className="chipRow">
                                {("freeShipping" in s && s.freeShipping) && <span className="signalPill ok">{t("rec_tag_free_shipping")}</span>}
                                <span className="signalPill neutral">{t("opens_in_new_tab")}</span>
                              </div>
                            </div>
                          </>
                        );

                        return canOpen ? (
                          <a
                            key={s.id}
                            href={s.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="fluid-card recCard"
                            style={{ textDecoration: "none", color: "inherit", cursor: "pointer" }}
                            aria-label={`${title} ${t("opens_in_new_tab")}`}
                          >
                            {card}
                          </a>
                        ) : (
                          <article key={s.id} className="fluid-card recCard">{card}</article>
                        );
                      })}
                    </div>
                    {bestFillCombo && bestFillCombo.items.length >= 2 && (
                      <p className="mutedText" style={{ marginTop: 8, marginBottom: 0 }}>
                        {t("fill_to_50_best_combo")}{" "}
                        {bestFillCombo.items.map((item, idx) => {
                          const label = `${item.title}`;
                          const suffix = idx < bestFillCombo.items.length - 1 ? " + " : "";
                          return item.url && isValidHttpUrl(item.url) ? (
                            <Fragment key={`${item.asin}-${idx}`}>
                              <a href={item.url} target="_blank" rel="noopener noreferrer">{label}</a>
                              {suffix}
                            </Fragment>
                          ) : (
                            <Fragment key={`${item.asin}-${idx}`}>
                              {label}
                              {suffix}
                            </Fragment>
                          );
                        })}
                        {` · $${bestFillCombo.total.toFixed(2)}`}
                      </p>
                    )}
                  </div>
                )}
              </>
            )}
            {result && submittedForm && (
              <div className="targetActions">
                <button className="secondary" type="button" onClick={addToWatchlist} disabled={addingWatchlist || watchlistAdded}>
                  {addingWatchlist ? t("loading") : watchlistAdded ? t("watchlist_added") : t("add_button")}
                </button>
                <span className="signalPill neutral" role="status">{t("home_alert_cta_disabled")}</span>
              </div>
            )}
          </div>
        </section>
      )}

      {(loading || (result && !result.free_shipping_country)) && (
        <>
          <section className="flowHint animate-flow-3">
            <span>{t("home_flow_hint")}</span>
          </section>

          <section className="recommendGrid animate-flow-3">
            {recommendations.map((r) => {
              const content = (
                <>
                  {r.type === "mock" ? (
                    <div className={`recImage ${r.cls}`} />
                  ) : (
                    r.imageUrl ? (
                      <img src={r.imageUrl} alt={r.title} className="recImage" style={{ objectFit: "cover" }} />
                    ) : (
                      <div className="recImage img-placeholder-1" />
                    )
                  )}
                  <h4>{r.type === "mock" ? t(r.titleKey) : r.title}</h4>
                  <div className="recMeta">
                    <strong>{r.price}</strong>
                    <div className="chipRow">
                      {r.type === "mock" && <span className="signalPill neutral">{t("home_demo_label")}</span>}
                      {r.type === "mock" ? (
                        <span className="signalPill ok">{t(r.tagKey)}</span>
                      ) : (
                        r.freeShipping && <span className="signalPill ok">{t("rec_tag_free_shipping")}</span>
                      )}
                    </div>
                  </div>
                </>
              );

              return r.type === "real" && r.url && isValidHttpUrl(r.url) ? (
                <a
                  key={r.id}
                  href={r.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label={`${r.title} ${t("opens_in_new_tab")}`}
                  className="fluid-card recCard"
                  style={{ textDecoration: "none", color: "inherit", cursor: "pointer" }}
                >
                  {content}
                </a>
              ) : (
                <article key={r.id} className="fluid-card recCard">
                  {content}
                </article>
              );
            })}
          </section>
        </>
      )}
    </main>
  );
}
