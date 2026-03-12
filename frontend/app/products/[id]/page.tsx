"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { getFallbackTrackedProducts, TrackedProduct } from "@/lib/watchlist";
import { useI18n } from "@/lib/i18n";

export default function ProductDetailsPage() {
  const { t } = useI18n();
  const { id } = useParams<{ id: string }>();
  const [items, setItems] = useState<TrackedProduct[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) {
      setLoading(false);
      return;
    }

    let cancelled = false;
    (async () => {
      try {
        const res = await fetch(`/api/v1/me/tracked-items/${encodeURIComponent(id)}`, { cache: "no-store" });
        if (!res.ok) throw new Error("failed");
        const body = await res.json();
        if (!cancelled) setItems(body.item ? [body.item] : []);
      } catch {
        if (!cancelled) setItems(getFallbackTrackedProducts());
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id]);

  const item = useMemo(() => items.find((x) => x.id === id), [items, id]);

  return (
    <main className="container">
      <section className="card">
        <Link href="/products">← {t("back_watchlist")}</Link>
      </section>
      <section className="card">
        {loading && <p>Loading…</p>}
        {!loading && !item && <p className="error">Tracked product not found.</p>}
        {item && (
          <>
            <h2>{t("details_title")}</h2>
            <ul>
              <li><strong>ID:</strong> {item.id}</li>
              <li><strong>URL:</strong> <a href={item.url} target="_blank" rel="noopener noreferrer">{item.url}</a></li>
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
