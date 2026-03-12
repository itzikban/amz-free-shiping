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

  const canSubmit = useMemo(() => {
    if (!form.url.startsWith("http")) return false;
    if (form.country === "US" && !form.zip.trim()) return false;
    return true;
  }, [form]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit) return;

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const params = new URLSearchParams({ url: form.url, country: form.country });
      if (form.country === "US" && form.zip.trim()) params.set("zip", form.zip.trim());
      const res = await fetch(`/api/check?${params.toString()}`);
      const body = await res.json();
      if (!res.ok) throw new Error(body?.error || "Request failed");
      setResult(body);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }

  async function addToWatchlist() {
    if (!canSubmit) return;
    try {
      const res = await fetch("/api/v1/me/tracked-items", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          url: form.url,
          country: form.country,
          zip: form.country === "US" ? form.zip.trim() : "",
        }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body?.error || t("err_add_watchlist"));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t("err_network"));
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
            <input
              className="clean-input"
              type="url"
              value={form.url}
              onChange={(e) => setForm((x) => ({ ...x, url: e.target.value }))}
              placeholder={t("home_url_placeholder")}
              required
            />
          </div>

          <div className="search-row fixed">
            <select
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
              <input
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
          <div className="targetImage img-placeholder-main" />
          <div className="targetBody">
            <h3>{t("home_target_analysis")}</h3>
            {error && <p className="error">{error}</p>}
            {result && (
              <>
                <p className="mutedText">{result.url}</p>
                <div className="targetMeta">
                  <span className="priceTag">{result.price_usd ? `$${result.price_usd.toFixed(2)}` : "-"}</span>
                  <span className={`signalPill ${result.free_shipping_country ? "ok" : "no"}`}>
                    {result.free_shipping_country ? t("free_for_destination") : t("not_free_for_destination")}
                  </span>
                </div>
                <div className="chipRow">
                  <span className="signalPill neutral">{result.country}</span>
                  <span className="signalPill neutral">{result.method}</span>
                  <span className="signalPill neutral">{result.signal}</span>
                </div>
              </>
            )}
            <div className="targetActions">
              <button className="secondary" type="button" onClick={addToWatchlist}>{t("add_button")}</button>
              <button className="btn-alert-pulse" type="button" disabled aria-disabled="true" title={t("home_alert_cta_disabled")}>
                {t("home_alert_cta_disabled")}
              </button>
            </div>
          </div>
        </section>
      )}

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
              <span className={`signalPill ${r.tagKey === "rec_tag_best_match" ? "neutral" : "ok"}`}>{t(r.tagKey)}</span>
            </div>
          </article>
        ))}
      </section>
    </main>
  );
}
