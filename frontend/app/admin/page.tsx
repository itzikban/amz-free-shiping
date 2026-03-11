"use client";

import { useEffect, useState } from "react";

type Metrics = {
  generated_at: string;
  monitors_total: number;
  monitors_running: number;
  monitors_stopped: number;
  notifications: number;
  user_tracked_items: number;
  user_alerts: number;
};

export default function AdminPage() {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [msg, setMsg] = useState<string>("");

  async function refresh() {
    const res = await fetch('/api/v1/admin/metrics');
    if (res.ok) setMetrics(await res.json());
  }

  async function runAction(path: string) {
    const res = await fetch(path, { method: 'POST' });
    const body = await res.json();
    setMsg(body?.message || body?.error || 'done');
    await refresh();
  }

  useEffect(() => { refresh(); }, []);

  return (
    <main className="container">
      <section className="card">
        <h1>Admin Operations (ITZ-19)</h1>
        <p className="mutedText">Backend metrics and operation actions.</p>
      </section>

      <section className="card">
        <h3>Metrics snapshot</h3>
        {!metrics && <p>Loading...</p>}
        {metrics && (
          <ul>
            <li>Monitors total: {metrics.monitors_total}</li>
            <li>Monitors running: {metrics.monitors_running}</li>
            <li>Monitors stopped: {metrics.monitors_stopped}</li>
            <li>Monitor notifications: {metrics.notifications}</li>
            <li>User tracked items: {metrics.user_tracked_items}</li>
            <li>User alerts: {metrics.user_alerts}</li>
            <li>Generated: {new Date(metrics.generated_at).toLocaleString()}</li>
          </ul>
        )}
      </section>

      <section className="card">
        <h3>Actions</h3>
        <div className="row">
          <button className="secondary" onClick={() => runAction('/api/v1/admin/actions/replay-failed-jobs')}>Replay failed jobs</button>
          <button className="secondary" onClick={() => runAction('/api/v1/admin/actions/retry-failed-notifications')}>Retry failed notifications</button>
        </div>
        {msg && <p className="mutedText">Result: {msg}</p>}
      </section>
    </main>
  );
}
