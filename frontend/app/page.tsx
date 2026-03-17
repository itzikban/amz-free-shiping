"use client";

import { useMemo, useState } from "react";
import type { CheckResponse } from "@/lib/types";
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
          // MT and CY don't need ZIP
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

  type RecommendationCard = {
    id: number;
    type: "mock" | "real";
    price: string;
  } & (
    | { type: "mock"; titleKey: string; cls: string }
    | { type: "real"; title: string; imageUrl?: string; url?: string }
  );

  const recommendations: RecommendationCard[] = result?.alternatives && result.alternatives.length > 0
    ? result.alternatives.map((alt, i) => ({
        id: i + 1,
        type: "real" as const,
        title: alt.title,
        price: alt.price_usd ? `$${alt.price_usd.toFixed(2)}` : "N/A",
        imageUrl: alt.image_url,
        url: alt.url,
      }))
    : mockRecommendations.map((r) => ({
        id: r.id,
        type: "mock" as const,
        titleKey: r.titleKey,
        price: r.price,
        cls: r.cls,
      }));

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
              <option value="NL">🇳🇱 Netherlands (NL)</option>
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

      {loading && (
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
                      <span className="signalPill ok">{t("rec_tag_free_shipping")}</span>
                    </div>
                  </div>
                </>
              );

              return r.type === "real" && r.url ? (
                <a
                  key={r.id}
                  href={r.url}
                  target="_blank"
                  rel="noopener noreferrer"
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
