"use client";

import { useEffect, useRef, useState } from "react";
import { useI18n } from "@/lib/i18n";

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
  const { t, formatDate } = useI18n();
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [msg, setMsg] = useState<string>("");
  const [metricsError, setMetricsError] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loginMsg, setLoginMsg] = useState("");
  const [loggingIn, setLoggingIn] = useState(false);
  const [authRequired, setAuthRequired] = useState(false);
  const [fetchMethod, setFetchMethod] = useState<"auto" | "http">("auto");
  const reqSeq = useRef(0);

  async function refresh() {
    const seq = ++reqSeq.current;
    try {
      const res = await fetch('/api/v1/admin/metrics');
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        if (seq === reqSeq.current) {
          setMetrics(null);
          setMetricsError(body?.error || t('admin_failed_load_metrics'));
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
        setMetricsError(t('admin_network_load_metrics'));
        setAuthRequired(false);
      }
    }
  }

  async function login() {
    setLoginMsg('');
    setLoggingIn(true);
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
      setLoginMsg(t('common_yes'));
      await refresh();
    } catch (err) {
      setLoginMsg(err instanceof Error ? err.message : 'network_error');
    } finally {
      setLoggingIn(false);
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
      setMsg(body?.message || body?.error || t('admin_action_completed'));
    } catch {
      setMsg(t('admin_network_action_error'));
    } finally {
      await refresh();
      setLoading(false);
    }
  }

  async function loadFetchMethod() {
    try {
      const res = await fetch('/api/v1/admin/fetch-method');
      if (res.ok) {
        const body = await res.json();
        setFetchMethod(body.method || 'auto');
      }
    } catch { /* ignore */ }
  }

  async function saveFetchMethod(method: "auto" | "http") {
    setFetchMethod(method);
    try {
      await fetch('/api/v1/admin/fetch-method', {
        method: 'PUT',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ method }),
      });
    } catch { /* ignore */ }
  }

  useEffect(() => {
    let active = true;
    (async () => {
      if (active) {
        await refresh();
        await loadFetchMethod();
      }
    })();
    return () => {
      active = false;
      reqSeq.current += 1;
    };
  }, []);

  return (
    <main className="container">
      <section className="card">
        <h1>{t('admin_title')} (ITZ-19)</h1>
        <p className="mutedText">{t('admin_subtitle')}</p>
      </section>

      <section className="card">
        <h3>{t('admin_metrics_snapshot')}</h3>
        {metricsError && <p className="mutedText">{metricsError}</p>}
        {authRequired && (
          <form className="row" style={{ marginBottom: 12 }} onSubmit={(e) => { e.preventDefault(); void login(); }}>
            <input aria-label={t('admin_login_username')} autoComplete="username" placeholder={t('admin_login_username')} value={username} onChange={(e) => setUsername(e.target.value)} />
            <input aria-label={t('admin_login_password')} autoComplete="current-password" placeholder={t('admin_login_password')} type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
            <button type="submit" className="secondary" disabled={loggingIn}>{loggingIn ? t('admin_logging_in') : t('admin_login_cta')}</button>
            {loginMsg && <span className="mutedText">{loginMsg}</span>}
          </form>
        )}
        {!metrics && !metricsError && <p>{t('admin_loading')}</p>}
        {metrics && (
          <ul>
            <li>{t('admin_monitors_total')}: {metrics.monitors_total}</li>
            <li>{t('admin_monitors_running')}: {metrics.monitors_running}</li>
            <li>{t('admin_monitors_stopped')}: {metrics.monitors_stopped}</li>
            <li>{t('admin_monitor_notifications')}: {metrics.notifications}</li>
            <li>{t('admin_user_tracked_items')}: {metrics.user_tracked_items}</li>
            <li>{t('admin_user_alerts')}: {metrics.user_alerts}</li>
            <li>{t('admin_generated')}: {formatDate(metrics.generated_at)}</li>
          </ul>
        )}
      </section>

      <section className="card">
        <h3>Fetch Method</h3>
        <p className="mutedText">Controls how product pages are fetched. Auto uses premium proxy when available.</p>
        <div className="fetchMethodToggle" style={{ justifyContent: 'flex-start', opacity: 1 }}>
          <label>
            <input type="radio" name="adminFetchMethod" value="auto" checked={fetchMethod === "auto"} onChange={() => saveFetchMethod("auto")} />
            Auto (proxy + HTTP)
          </label>
          <label>
            <input type="radio" name="adminFetchMethod" value="http" checked={fetchMethod === "http"} onChange={() => saveFetchMethod("http")} />
            HTTP only
          </label>
        </div>
      </section>

      <section className="card">
        <h3>{t('admin_actions')}</h3>
        <div className="row">
          <button className="secondary" disabled={loading} onClick={() => runAction('/api/v1/admin/actions/replay-failed-jobs')}>
            {t('admin_replay_failed_jobs')}
          </button>
          <button className="secondary" disabled={loading} onClick={() => runAction('/api/v1/admin/actions/retry-failed-notifications')}>
            {t('admin_retry_failed_notifications')}
          </button>
        </div>
        {msg && <p className="mutedText">{t('admin_result_prefix')}: {msg}</p>}
      </section>
    </main>
  );
}
