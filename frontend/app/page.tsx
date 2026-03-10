"use client";

import { useEffect, useMemo, useState } from "react";
import type { CheckResponse } from "@/lib/types";

type FormState = {
  url: string;
  country: "US" | "IL";
  zip: string;
};

const SAMPLE = "https://www.amazon.com/dp/B0DHCZBKW7";

export default function HomePage() {
  const [form, setForm] = useState<FormState>({ url: SAMPLE, country: "US", zip: "10013" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<CheckResponse | null>(null);
  const [health, setHealth] = useState<"checking" | "up" | "down">("checking");

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
      const res = await fetch(`/api/check?${params.toString()}`, { method: "GET" });
      const body = await res.json();
      if (!res.ok) throw new Error(body?.error || "Request failed");
      setResult(body);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetch("/api/health")
      .then((r) => (r.ok ? setHealth("up") : setHealth("down")))
      .catch(() => setHealth("down"));
  }, []);

  const isFree = result?.free_shipping_country;
  const isGenericFree = result?.free_shipping;

  return (
    <main className="container">
      <section className="hero card">
        <h1>AMZ Free Shipping Checker</h1>
        <p>
          Country-aware shipping checker powered by your backend. Enter an Amazon URL, destination,
          and get a strict free-shipping verdict.
        </p>
        <div className={`health ${health}`}>
          Backend: {health === "checking" ? "Checking..." : health === "up" ? "Online" : "Offline"}
        </div>
      </section>

      <section className="card">
        <form onSubmit={onSubmit} className="form">
          <label>
            Product URL
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
              Destination country
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
              ZIP (US only)
              <input
                type="text"
                value={form.zip}
                onChange={(e) => setForm((x) => ({ ...x, zip: e.target.value }))}
                placeholder="10013"
                disabled={form.country !== "US"}
              />
            </label>
          </div>

          <button disabled={!canSubmit || loading}>
            {loading ? "Checking…" : "Check free shipping"}
          </button>
        </form>
      </section>

      {(error || result) && (
        <section className="card result">
          {error && <p className="error">Error: {error}</p>}

          {result && (
            <>
              <div className={`badge ${isFree ? "ok" : "no"}`}>
                {isFree ? "✅ Free shipping for destination" : "❌ Not free for destination"}
              </div>
              {!isFree && isGenericFree && (
                <p className="hint">Generic free-shipping text was found, but not confirmed for selected destination.</p>
              )}
              <ul>
                <li>
                  <strong>Country:</strong> {result.country}
                </li>
                <li>
                  <strong>free_shipping (generic):</strong> {String(result.free_shipping)}
                </li>
                <li>
                  <strong>free_shipping_country (strict):</strong> {String(result.free_shipping_country)}
                </li>
                <li>
                  <strong>Signal:</strong> {result.signal}
                </li>
                <li>
                  <strong>Method:</strong> {result.method}
                </li>
                <li>
                  <strong>Checked:</strong> {new Date(result.checked_at).toLocaleString()}
                </li>
              </ul>
              <details>
                <summary>Raw backend response</summary>
                <pre>{JSON.stringify(result, null, 2)}</pre>
              </details>
            </>
          )}
        </section>
      )}
    </main>
  );
}
