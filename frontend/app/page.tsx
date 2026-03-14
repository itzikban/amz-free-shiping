"use client";

import { useMemo, useState } from "react";
import type { CheckResponse } from "@/lib/types";
import { useI18n } from "@/lib/i18n";

type FormState = {
  url: string;
  country: "US" | "IL";
  zip: string;
};

const SAMPLE = "https://www.amazon.com/dp/B0DHCZBKW7";

export default function HomePage() {
  const { t } = useI18n();
  const [form, setForm] = useState<FormState>({ url: SAMPLE, country: "IL", zip: "10013" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<CheckResponse | null>(null);
  const [submittedForm, setSubmittedForm] = useState<FormState | null>(null);
  const [addingWatchlist, setAddingWatchlist] = useState(false);
  const [watchlistAdded, setWatchlistAdded] = useState(false);
  const [fetchMethod, setFetchMethod] = useState<"auto" | "http">("auto");

  const canSubmit = useMemo(() => {
    if (!form.url.startsWith("http")) return false;
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

  const recommendations = [
    { id: 1, titleKey: "rec_title_1", price: "$345", tagKey: "rec_tag_free_shipping", cls: "img-placeholder-1" },
    { id: 2, titleKey: "rec_title_2", price: "$348", tagKey: "rec_tag_best_match", cls: "img-placeholder-2" },
    { id: 3, titleKey: "rec_title_3", price: "$429", tagKey: "rec_tag_free_shipping", cls: "img-placeholder-3" },
  ] as const;

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
              onChange={(e) => setForm((x) => ({ ...x, country: e.target.value as "US" | "IL" }))}
            >
              <option value="IL">{t("home_country_il")}</option>
              <option value="US">{t("home_country_us")}</option>
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

        <div className="fetchMethodToggle">
          <label>
            <input
              type="radio"
              name="fetchMethod"
              value="auto"
              checked={fetchMethod === "auto"}
              onChange={() => setFetchMethod("auto")}
            />
            Decodo + HTTP
          </label>
          <label>
            <input
              type="radio"
              name="fetchMethod"
              value="http"
              checked={fetchMethod === "http"}
              onChange={() => setFetchMethod("http")}
            />
            HTTP only
          </label>
        </div>
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
                  <span className="signalPill neutral">{result.method}</span>
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
            {recommendations.map((r) => (
              <article key={r.id} className="fluid-card recCard">
                <div className={`recImage ${r.cls}`} />
                <h4>{t(r.titleKey)}</h4>
                <div className="recMeta">
                  <strong>{r.price}</strong>
                  <div className="chipRow">
                    <span className="signalPill neutral">{t("home_demo_label")}</span>
                    <span className={`signalPill ${r.tagKey === "rec_tag_best_match" ? "neutral" : "ok"}`}>{t(r.tagKey)}</span>
                  </div>
                </div>
              </article>
            ))}
          </section>
        </>
      )}
    </main>
  );
}
