"use client";

import { useEffect, useMemo, useState } from "react";
import { FALLBACK_ALERTS, UserAlert } from "@/lib/watchlist";

export default function AlertsPage() {
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
        if (!cancelled) setAlerts(FALLBACK_ALERTS);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const filtered = useMemo(() => {
    if (filter === "all") return alerts;
    if (filter === "free") return alerts.filter((x) => x.message.toLowerCase().includes("free shipping"));
    return alerts.filter((x) => x.message.toLowerCase().includes("price"));
  }, [alerts, filter]);

  return (
    <main className="container">
      <section className="card hero">
        <h1>Alerts Center</h1>
        <p>In-app alerts for shipping and price changes.</p>
        <div className="row" style={{ gridTemplateColumns: "repeat(3, 1fr)" }}>
          <button className={filter === "all" ? "" : "secondary"} onClick={() => setFilter("all")}>All</button>
          <button className={filter === "free" ? "" : "secondary"} onClick={() => setFilter("free")}>Free shipping</button>
          <button className={filter === "price" ? "" : "secondary"} onClick={() => setFilter("price")}>Price changes</button>
        </div>
      </section>
      <section className="card">
        {loading ? <p>Loading…</p> : (
          <ul>
            {filtered.length === 0 && <li>No alerts match current filter.</li>}
            {filtered.map((a) => (
              <li key={a.id} className="monitorItem">
                <div>🔔 {a.message}</div>
                <small>{new Date(a.created_at).toLocaleString()}</small>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
