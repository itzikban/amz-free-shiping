"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { FALLBACK_TRACKED_PRODUCTS, TrackedProduct } from "@/lib/watchlist";

export default function ProductDetailsPage({ params }: { params: { id: string } }) {
  const [items, setItems] = useState<TrackedProduct[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch("/api/v1/me/tracked-items", { cache: "no-store" });
        if (!res.ok) throw new Error("failed");
        const body = await res.json();
        if (!cancelled) setItems(body.items || []);
      } catch {
        if (!cancelled) setItems(FALLBACK_TRACKED_PRODUCTS);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const item = useMemo(() => items.find((x) => x.id === params.id), [items, params.id]);

  return (
    <main className="container">
      <section className="card">
        <Link href="/products">← Back to watchlist</Link>
      </section>
      <section className="card">
        {loading && <p>Loading…</p>}
        {!loading && !item && <p className="error">Tracked product not found.</p>}
        {item && (
          <>
            <h2>Tracked Product Details</h2>
            <ul>
              <li><strong>ID:</strong> {item.id}</li>
              <li><strong>URL:</strong> <a href={item.url} target="_blank">{item.url}</a></li>
              <li><strong>Country:</strong> {item.country}</li>
              <li><strong>ZIP:</strong> {item.zip || "-"}</li>
              <li><strong>Price:</strong> {item.last_price_usd ? `$${item.last_price_usd.toFixed(2)}` : "-"}</li>
              <li><strong>Free shipping:</strong> {String(item.free_shipping_country)}</li>
              <li><strong>Signal:</strong> {item.signal}</li>
              <li><strong>Method:</strong> {item.method || "-"}</li>
              <li><strong>Last checked:</strong> {new Date(item.last_checked_at).toLocaleString()}</li>
            </ul>
          </>
        )}
      </section>
    </main>
  );
}
