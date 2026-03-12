"use client";

import { useEffect, useMemo, useState } from "react";
import { getFallbackAlerts, UserAlert } from "@/lib/watchlist";
import { useI18n } from "@/lib/i18n";

export default function AlertsPage() {
  const { t } = useI18n();
  const [alerts, setAlerts] = useState<UserAlert[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<"all" | "free" | "price">("all");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch("/api/v1/me/alerts", { cache: "no-store" });
        if (!res.ok) throw new Error("failed");
        const body = await res.json();
        if (!cancelled) setAlerts(body.alerts || []);
      } catch {
        if (!cancelled) setAlerts(getFallbackAlerts());
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const filtered = useMemo(() => {
    if (filter === "all") return alerts;
    if (filter === "free") return alerts.filter((x) => x.type === "free_shipping");
    return alerts.filter((x) => x.type === "price_change");
  }, [alerts, filter]);

  return (
    <main className="container">
      <section className="card hero">
        <h1>{t("alerts_title")}</h1>
        <p>{t("alerts_subtitle")}</p>

        <div className="chipRow" style={{ marginTop: 12 }}>
          <button className={filter === "all" ? "" : "secondary"} onClick={() => setFilter("all")}>
            {t("alerts_filter_all")}
          </button>
          <button className={filter === "free" ? "" : "secondary"} onClick={() => setFilter("free")}>
            {t("alerts_filter_free")}
          </button>
          <button className={filter === "price" ? "" : "secondary"} onClick={() => setFilter("price")}>
            {t("alerts_filter_price")}
          </button>
        </div>
      </section>

      <section className="card">
        {loading ? (
          <p>{t("loading")}</p>
        ) : (
          <ul className="listClean">
            {filtered.length === 0 && <li>{t("alerts_empty_filtered")}</li>}
            {filtered.map((a) => (
              <li key={a.id} className="timelineItem">
                <div className="timelineDot" />
                <div>
                  <small>{new Date(a.created_at).toLocaleString()}</small>
                  <div>{a.message}</div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
