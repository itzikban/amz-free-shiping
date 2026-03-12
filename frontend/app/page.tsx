"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import type { CheckResponse } from "@/lib/types";
import { useI18n } from "@/lib/i18n";

type FormState = {
  url: string;
  country: "US" | "IL";
  zip: string;
};

type TrackedItem = {
  id: string;
  url: string;
  country: string;
  free_shipping_country: boolean;
  last_checked_at: string;
};

type Alert = {
  id: string;
  message: string;
  created_at: string;
};

const SAMPLE = "https://www.amazon.com/dp/B0DHCZBKW7";

export default function HomePage() {
  const { t } = useI18n();
  const [form, setForm] = useState<FormState>({ url: SAMPLE, country: "US", zip: "10013" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<CheckResponse | null>(null);
  const [health, setHealth] = useState<"checking" | "up" | "down">("checking");
  const [items, setItems] = useState<TrackedItem[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);

  const canSubmit = useMemo(() => {
    if (!form.url.startsWith("http")) return false;
    if (form.country === "US" && !form.zip.trim()) return false;
    return true;
  }, [form]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [h, i, a] = await Promise.all([
          fetch("/api/health"),
          fetch("/api/v1/me/tracked-items"),
          fetch("/api/v1/me/alerts"),
        ]);

        if (!cancelled) {
          setHealth(h.ok ? "up" : "down");
          if (i.ok) {
            const b = await i.json();
            setItems((b.items || []).slice(0, 3));
          }
          if (a.ok) {
            const b = await a.json();
            setAlerts((b.alerts || []).slice(0, 3));
          }
        }
      } catch {
        if (!cancelled) setHealth("down");
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

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
      setHealth("up");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
      setHealth("down");
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
        setError(body?.error || t("err_add_watchlist"));
        return;
      }
      const i = await fetch("/api/v1/me/tracked-items");
      if (i.ok) {
        const b = await i.json();
        setItems((b.items || []).slice(0, 3));
      }
      setError(null);
    } catch {
      setError(t("err_network"));
    }
  }

  return (
    <main className="container">
      <section className="card hero">
        <h1>{t("app_title")}</h1>
        <p>{t("app_subtitle")}</p>
        <div className={`health ${health}`}>
          {t("backend_status")}: {health === "checking" ? t("checking") : health === "up" ? t("online") : t("offline")}
        </div>
      </section>

      <section className="card">
        <h3>{t("quick_check")}</h3>
        <form onSubmit={onSubmit} className="form">
          <label>
            {t("product_url")}
            <input
              type="url"
              value={form.url}
              onChange={(e) => setForm((x) => ({ ...x, url: e.target.value }))}
              placeholder="https://www.amazon.com/dp/..."
              required
            />
          </label>

          <div className="row">
            <label>
              {t("destination_country")}
              <select
                value={form.country}
                onChange={(e) =>
                  setForm((x) => ({ ...x, country: e.target.value as "US" | "IL", zip: e.target.value === "US" ? x.zip : "" }))
                }
              >
                <option value="US">United States</option>
                <option value="IL">Israel</option>
              </select>
            </label>

            <label>
              {t("zip_us")}
              <input
                type="text"
                value={form.zip}
                onChange={(e) => setForm((x) => ({ ...x, zip: e.target.value }))}
                placeholder="10013"
                disabled={form.country !== "US"}
              />
            </label>
          </div>

          <div className="row">
            <button disabled={!canSubmit || loading}>{loading ? t("loading") : t("check_button")}</button>
            <button type="button" className="secondary" disabled={!canSubmit} onClick={addToWatchlist}>
              {t("add_button")}
            </button>
          </div>
        </form>
      </section>

      {(error || result) && (
        <section className="card result">
          {error && <p className="error">{error}</p>}
          {result && (
            <>
              <div className={`badge ${result.free_shipping_country ? "ok" : "no"}`}>
                {result.free_shipping_country ? t("free_for_destination") : t("not_free_for_destination")}
              </div>
              <ul>
                <li><strong>{t("country_label")}:</strong> {result.country}</li>
                <li><strong>{t("price_label")}:</strong> {result.price_usd ? `$${result.price_usd.toFixed(2)}` : "-"}</li>
                <li><strong>{t("signal_label")}:</strong> {result.signal}</li>
                <li><strong>{t("method_label")}:</strong> {result.method}</li>
              </ul>
            </>
          )}
        </section>
      )}

      <section className="row split">
        <div className="card">
          <div className="actionsRow">
            <h3>{t("watchlist_title")}</h3>
            <Link href="/products">{t("view_all")}</Link>
          </div>
          <ul>
            {items.length === 0 && <li>{t("empty_watchlist")}</li>}
            {items.map((it) => (
              <li key={it.id} className="monitorItem">
                <div>
                  <strong>{it.country}</strong> · {it.free_shipping_country ? "✅" : "❌"}
                  <br />
                  <small>{it.url}</small>
                </div>
              </li>
            ))}
          </ul>
        </div>

        <div className="card">
          <div className="actionsRow">
            <h3>{t("alerts_title")}</h3>
            <Link href="/alerts">{t("view_all")}</Link>
          </div>
          <ul>
            {alerts.length === 0 && <li>{t("empty_alerts")}</li>}
            {alerts.map((a) => (
              <li key={a.id} className="monitorItem">
                <div>
                  <small>{new Date(a.created_at).toLocaleString()}</small>
                  <br />
                  {a.message}
                </div>
              </li>
            ))}
          </ul>
        </div>
      </section>
    </main>
  );
}
