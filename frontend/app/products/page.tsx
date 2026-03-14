"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { getFallbackTrackedProducts, TrackedProduct } from "@/lib/watchlist";
import { useI18n } from "@/lib/i18n";

/**
 * Render the products watchlist page with KPIs, search, and a list of tracked products.
 *
 * Fetches tracked products on mount and displays fallback data if the network request fails.
 *
 * @returns The React element for the products watchlist page.
 */
export default function ProductsPage() {
  const { t, formatDate } = useI18n();
  const [items, setItems] = useState<TrackedProduct[]>([]);
  const [loading, setLoading] = useState(true);
  const [usingFallback, setUsingFallback] = useState(false);
  const [query, setQuery] = useState("");

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

  const counts = useMemo(
    () => ({
      total: items.length,
      free: items.filter((x) => x.free_shipping_country).length,
      notFree: items.filter((x) => !x.free_shipping_country).length,
    }),
    [items]
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return items;
    return items.filter((it) => it.url.toLowerCase().includes(q) || it.country.toLowerCase().includes(q));
  }, [items, query]);

  return (
    <main className="container">
      <section className="card hero">
        <h1>{t("watchlist_title")}</h1>
        <p>{t("products_subtitle")}</p>

        <div className="statGrid">
          <div className="kpiCard">
            <span>{t("products_total_tracked")}</span>
            <strong>{counts.total}</strong>
          </div>
          <div className="kpiCard ok">
            <span>{t("products_free_shipping")}</span>
            <strong>{counts.free}</strong>
          </div>
          <div className="kpiCard no">
            <span>{t("products_not_free")}</span>
            <strong>{counts.notFree}</strong>
          </div>
        </div>

        {usingFallback && <p className="hint">{t("products_fallback_notice")}</p>}
      </section>

      <section className="card">
        <div className="actionsRow">
          <h3>{t("products_heading")}</h3>
          <Link href="/">{t("products_add_from_checker")}</Link>
        </div>

        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t("products_search_placeholder")}
          style={{ marginBottom: 12 }}
        />

        {loading ? (
          <p>{t("loading")}</p>
        ) : (
          <ul className="listClean">
            {filtered.length === 0 && <li>{query.trim() ? t("products_no_results") : t("products_empty")}</li>}
            {filtered.map((it) => (
              <li key={it.id} className="monitorItem">
                {it.image_url && (
                  <img className="watchlistThumb" src={it.image_url} alt={it.title || "Product"} />
                )}
                <div className="monitorItemBody">
                  {it.title && <strong className="watchlistTitle">{it.title}</strong>}
                  <div className="chipRow">
                    <span className={`signalPill ${it.free_shipping_country ? "ok" : "no"}`}>
                      {it.free_shipping_country ? t("products_status_free") : t("products_status_not_free")}
                    </span>
                    <span className="signalPill neutral">{it.country}</span>
                  </div>
                  <small>{it.url}</small>
                  <br />
                  <small>
                    {t("products_last_checked")} {formatDate(it.last_checked_at)}
                  </small>
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
