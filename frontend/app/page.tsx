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
    { id: 1, title: "Sony WH-1000XM5 (International Version)", price: "$345", tag: "FREE SHIPPING", cls: "img-placeholder-1" },
    { id: 2, title: "Sony WH-1000XM5 Wireless - Silver", price: "$348", tag: "BEST MATCH", cls: "img-placeholder-2" },
    { id: 3, title: "Bose QuietComfort Ultra - Black", price: "$429", tag: "FREE SHIPPING", cls: "img-placeholder-3" },
  ];

  return (
    <main className="container nexusHome">
      <section className="heroCenter animate-flow-1">
        <h1>Discover Global Shipping</h1>
        <p>{t("app_subtitle")}</p>

        <form className="search-container" onSubmit={onSubmit}>
          <div className="search-row grow">
            <input
              className="clean-input"
              type="url"
              value={form.url}
              onChange={(e) => setForm((x) => ({ ...x, url: e.target.value }))}
              placeholder="Paste Amazon URL here..."
              required
            />
          </div>

          <div className="search-row fixed">
            <select
              className="clean-input"
              value={form.country}
              onChange={(e) => setForm((x) => ({ ...x, country: e.target.value as "US" | "IL" }))}
            >
              <option value="IL">Israel (IL)</option>
              <option value="US">United States (US)</option>
            </select>
          </div>

          {form.country === "US" && (
            <div className="search-row fixed">
              <input
                className="clean-input"
                type="text"
                value={form.zip}
                onChange={(e) => setForm((x) => ({ ...x, zip: e.target.value }))}
                placeholder="ZIP"
              />
            </div>
          )}

          <button className="analyzeBtn" disabled={!canSubmit || loading}>
            {loading ? t("loading") : "Analyze"}
          </button>
        </form>
      </section>

      {(result || error) && (
        <section className="fluid-card targetCard animate-flow-2">
          <div className="targetImage img-placeholder-main" />
          <div className="targetBody">
            <h3>Target Analysis</h3>
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
              <button className="btn-alert-pulse" type="button">Alert me on Free Shipping</button>
            </div>
          </div>
        </section>
      )}

      <section className="flowHint animate-flow-3">
        <span>While you wait, we found alternatives with free shipping</span>
      </section>

      <section className="recommendGrid animate-flow-3">
        {recommendations.map((r) => (
          <article key={r.id} className="fluid-card recCard">
            <div className={`recImage ${r.cls}`} />
            <h4>{r.title}</h4>
            <div className="recMeta">
              <strong>{r.price}</strong>
              <span className={`signalPill ${r.tag === "BEST MATCH" ? "neutral" : "ok"}`}>{r.tag}</span>
            </div>
          </article>
        ))}
      </section>
    </main>
  );
}
