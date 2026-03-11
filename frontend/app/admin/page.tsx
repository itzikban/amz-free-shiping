"use client";

import { useEffect, useRef, useState } from "react";

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
  const [metricsError, setMetricsError] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loginMsg, setLoginMsg] = useState("");
  const [authRequired, setAuthRequired] = useState(false);
  const reqSeq = useRef(0);

  async function refresh() {
    const seq = ++reqSeq.current;
    try {
      const res = await fetch('/api/v1/admin/metrics');
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        if (seq === reqSeq.current) {
          setMetrics(null);
          setMetricsError(body?.error || 'Failed to load metrics');
          setAuthRequired(res.status === 401 || res.status === 403);
        }
        return;
      }
      const data = await res.json();
      if (seq === reqSeq.current) {
        setMetrics(data);
        setMetricsError('');
        setAuthRequired(false);
      }
    } catch {
      if (seq === reqSeq.current) {
        setMetrics(null);
        setMetricsError('Network error while loading metrics');
        setAuthRequired(false);
      }
    }
  }

  async function login() {
    setLoginMsg('');
    try {
      const res = await fetch('/api/v1/admin/login', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        setLoginMsg(body?.error || `login_failed_${res.status}`);
        return;
      }
      setLoginMsg('login_ok');
      await refresh();
    } catch (err) {
      setLoginMsg(err instanceof Error ? err.message : 'network_error');
    }
  }

  async function runAction(path: string) {
    setLoading(true);
    try {
      const res = await fetch(path, { method: 'POST' });
      if (!res.ok) {
        const fallback = await res.text().catch(() => '');
        setMsg(`Action failed (${res.status})${fallback ? `: ${fallback.slice(0, 160)}` : ''}`);
        return;
      }
      const body = await res.json().catch(() => ({}));
      setMsg(body?.message || body?.error || 'Action completed');
    } catch {
      setMsg('Network or parse error while running action');
    } finally {
      await refresh();
      setLoading(false);
    }
  }

  useEffect(() => {
    let active = true;
    (async () => {
      if (active) await refresh();
    })();
    return () => {
      active = false;
      reqSeq.current += 1;
    };
  }, []);

  return (
    <main className="container">
      <section className="card">
        <h1>Admin Operations (ITZ-19)</h1>
        <p className="mutedText">Backend metrics and operation actions.</p>
      </section>

      <section className="card">
        <h3>Metrics snapshot</h3>
        {metricsError && <p className="mutedText">{metricsError}</p>}
        {authRequired && (
          <div className="row" style={{ marginBottom: 12 }}>
            <input aria-label="Admin username" placeholder="admin username" value={username} onChange={(e) => setUsername(e.target.value)} />
            <input aria-label="Admin password" placeholder="admin password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
            <button className="secondary" onClick={login}>Login</button>
            {loginMsg && <span className="mutedText">{loginMsg}</span>}
          </div>
        )}
        {!metrics && !metricsError && <p>Loading...</p>}
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
          <button className="secondary" disabled={loading} onClick={() => runAction('/api/v1/admin/actions/replay-failed-jobs')}>
            Replay failed jobs
          </button>
          <button className="secondary" disabled={loading} onClick={() => runAction('/api/v1/admin/actions/retry-failed-notifications')}>
            Retry failed notifications
          </button>
        </div>
        {msg && <p className="mutedText">Result: {msg}</p>}
      </section>
    </main>
  );
}
