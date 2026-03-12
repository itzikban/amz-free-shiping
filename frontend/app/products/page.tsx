"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { getFallbackTrackedProducts, TrackedProduct } from "@/lib/watchlist";
import { useI18n } from "@/lib/i18n";

export default function ProductsPage() {
  const { t } = useI18n();
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
        <h1>{t("watchlist_title")}</h1>
        <p>{t("products_subtitle")}</p>
        <div className="row" style={{ gridTemplateColumns: "repeat(3, 1fr)" }}>
          <div className="card"><strong>{items.length}</strong><br /><small>{t("products_total_tracked")}</small></div>
          <div className="card"><strong>{counts.free}</strong><br /><small>{t("products_free_shipping")}</small></div>
          <div className="card"><strong>{counts.notFree}</strong><br /><small>{t("products_not_free")}</small></div>
        </div>
        {usingFallback && <p className="hint">{t("products_fallback_notice")}</p>}
      </section>

      <section className="card">
        <div className="row actionsRow">
          <h3>{t("products_heading")}</h3>
          <Link href="/">{t("products_add_from_checker")}</Link>
        </div>

        {loading ? <p>{t("loading")}</p> : (
          <ul>
            {items.length === 0 && <li>{t("products_empty")}</li>}
            {items.map((it) => (
              <li key={it.id} className="monitorItem">
                <div>
                  <strong>{it.country}</strong> · {it.free_shipping_country ? t("products_status_free") : t("products_status_not_free")}
                  <br />
                  <small>{it.url}</small>
                  <br />
                  <small>{t("products_last_checked")} {new Date(it.last_checked_at).toLocaleString()}</small>
                </div>
                <div>
                  <Link href={`/products/${encodeURIComponent(it.id)}`}>{t("products_view_details")}</Link>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
