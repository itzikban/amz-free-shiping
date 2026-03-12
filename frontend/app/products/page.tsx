"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { getFallbackTrackedProducts, TrackedProduct } from "@/lib/watchlist";

export default function ProductsPage() {
  const [items, setItems] = useState<TrackedProduct[]>([]);
  const [loading, setLoading] = useState(true);
  const [usingFallback, setUsingFallback] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch("/api/v1/me/tracked-items", { cache: "no-store" });
        if (!res.ok) throw new Error("failed");
        const body = await res.json();
        if (!cancelled) {
          setItems(body.items || []);
          setUsingFallback(false);
        }
      } catch {
        if (!cancelled) {
          setItems(getFallbackTrackedProducts());
          setUsingFallback(true);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const counts = useMemo(() => ({
    free: items.filter((x) => x.free_shipping_country).length,
    notFree: items.filter((x) => !x.free_shipping_country).length,
  }), [items]);

  return (
    <main className="container">
      <section className="card hero">
        <h1>Watchlist</h1>
        <p>Tracked products with destination-specific shipping verdict.</p>
        <div className="row" style={{ gridTemplateColumns: "repeat(3, 1fr)" }}>
          <div className="card"><strong>{items.length}</strong><br /><small>Total tracked</small></div>
          <div className="card"><strong>{counts.free}</strong><br /><small>Free shipping</small></div>
          <div className="card"><strong>{counts.notFree}</strong><br /><small>Not free</small></div>
        </div>
        {usingFallback && <p className="hint">Backend unavailable — showing local fallback sample data.</p>}
      </section>

      <section className="card">
        <div className="row actionsRow">
          <h3>Tracked products</h3>
          <Link href="/">+ Add from checker</Link>
        </div>

        {loading ? <p>Loading…</p> : (
          <ul>
            {items.length === 0 && <li>No tracked products yet.</li>}
            {items.map((it) => (
              <li key={it.id} className="monitorItem">
                <div>
                  <strong>{it.country}</strong> · {it.free_shipping_country ? "✅ Free" : "❌ Not free"}
                  <br />
                  <small>{it.url}</small>
                  <br />
                  <small>Last checked: {new Date(it.last_checked_at).toLocaleString()}</small>
                </div>
                <div>
                  <Link href={`/products/${encodeURIComponent(it.id)}`}>View details</Link>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
