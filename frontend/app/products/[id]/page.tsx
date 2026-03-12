"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { getFallbackTrackedProducts, TrackedProduct } from "@/lib/watchlist";
import { useI18n } from "@/lib/i18n";

export default function ProductDetailsPage() {
  const { t, formatDate } = useI18n();
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
        {loading && <p>{t("loading")}</p>}
        {!loading && !item && <p className="error">{t("details_not_found")}</p>}
        {item && (
          <>
            <h2>{t("details_title")}</h2>
            <ul>
              <li><strong>{t("details_id")}</strong> {item.id}</li>
              <li><strong>{t("details_url")}</strong> <a href={item.url} target="_blank" rel="noopener noreferrer">{item.url}</a></li>
              <li><strong>{t("details_country")}</strong> {item.country}</li>
              <li><strong>{t("details_zip")}</strong> {item.zip || t("common_dash")}</li>
              <li><strong>{t("details_price")}</strong> {item.last_price_usd ? `$${item.last_price_usd.toFixed(2)}` : t("common_dash")}</li>
              <li><strong>{t("details_free_shipping")}</strong> {item.free_shipping_country ? t("common_yes") : t("common_no")}</li>
              <li><strong>{t("details_signal")}</strong> {item.signal}</li>
              <li><strong>{t("details_method")}</strong> {item.method || t("common_dash")}</li>
              <li><strong>{t("details_last_checked")}</strong> {formatDate(item.last_checked_at)}</li>
            </ul>
          </>
        )}
      </section>
    </main>
  );
}
