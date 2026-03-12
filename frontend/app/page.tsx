"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import type { CheckResponse } from "@/lib/types";
import { useI18n } from "@/lib/i18n";

type FormState = {
  url: string;
  country: "US" | "IL";
  zip: string;
};

type Monitor = {
  id: string;
  url: string;
  country: string;
  zip?: string;
  interval_seconds: number;
  max_runs: number;
  runs_done: number;
  running: boolean;
  last_checked_at?: string;
  last_status?: boolean;
  last_signal?: string;
  last_method?: string;
  last_price_usd?: number;
  history: Array<{
    at: string;
    price_usd?: number;
    free_shipping: boolean;
    free_shipping_country: boolean;
    signal: string;
    method: string;
  }>;
};

type Notification = { monitor_id: string; at: string; message: string };
type Me = { id: string; name: string };
type UserTrackedItem = {
  id: string;
  url: string;
  country: string;
  zip?: string;
  last_checked_at: string;
  last_price_usd?: number;
  free_shipping_country: boolean;
  signal: string;
};
type UserAlert = { id: string; message: string; created_at: string };
type InAppNotification = { id: string; title: string; message: string; read: boolean; created_at: string; read_at?: string };
type NotificationPreferences = { in_app_enabled: boolean; on_item_added: boolean };

const SAMPLE = "https://www.amazon.com/dp/B0DHCZBKW7";

export default function HomePage() {
  const { t } = useI18n();
  const [form, setForm] = useState<FormState>({ url: SAMPLE, country: "US", zip: "10013" });
  const [intervalSec, setIntervalSec] = useState(20);
  const [loading, setLoading] = useState(false);
  const [maxRuns, setMaxRuns] = useState(10);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<CheckResponse | null>(null);
  const [health, setHealth] = useState<"checking" | "up" | "down">("checking");
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [me, setMe] = useState<Me | null>(null);
  const [userItems, setUserItems] = useState<UserTrackedItem[]>([]);
  const [userAlerts, setUserAlerts] = useState<UserAlert[]>([]);
  const [inAppNotifications, setInAppNotifications] = useState<InAppNotification[]>([]);
  const [notificationPrefs, setNotificationPrefs] = useState<NotificationPreferences>({ in_app_enabled: true, on_item_added: true });
  const refreshSeqRef = useRef(0);

  const canSubmit = useMemo(() => {
    if (!form.url.startsWith("http")) return false;
    if (form.country === "US" && !form.zip.trim()) return false;
    return true;
  }, [form]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit) return;

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const params = new URLSearchParams({ url: form.url, country: form.country });
      if (form.country === "US" && form.zip.trim()) params.set("zip", form.zip.trim());
      const res = await fetch(`/api/check?${params.toString()}`, { method: "GET" });
      const body = await res.json();
      if (!res.ok) throw new Error(body?.error || "Request failed");
      setResult(body);
      setHealth("up");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
      setHealth("down");
    } finally {
      setLoading(false);
    }
  }

  async function refreshUserPanel() {
    const seq = ++refreshSeqRef.current;
    const [meRes, itemsRes, alertsRes, notificationsRes, prefsRes] = await Promise.allSettled([
      fetch('/api/v1/me'),
      fetch('/api/v1/me/tracked-items'),
      fetch('/api/v1/me/alerts'),
      fetch('/api/v1/me/notifications'),
      fetch('/api/v1/me/notification-preferences'),
    ]);

    if (seq !== refreshSeqRef.current) return;

    if (meRes.status === 'fulfilled' && meRes.value.ok) {
      const meBody = await meRes.value.json();
      if (seq === refreshSeqRef.current) setMe(meBody);
    }
    if (itemsRes.status === 'fulfilled' && itemsRes.value.ok) {
      const b = await itemsRes.value.json();
      if (seq === refreshSeqRef.current) setUserItems(b.items || []);
    }
    if (alertsRes.status === 'fulfilled' && alertsRes.value.ok) {
      const b = await alertsRes.value.json();
      if (seq === refreshSeqRef.current) setUserAlerts(b.alerts || []);
    }

    if (notificationsRes.status === 'fulfilled' && notificationsRes.value.ok) {
      const b = await notificationsRes.value.json();
      if (seq === refreshSeqRef.current) setInAppNotifications(b.notifications || []);
    }
    if (prefsRes.status === 'fulfilled' && prefsRes.value.ok) {
      const b = await prefsRes.value.json();
      if (seq === refreshSeqRef.current) {
        setNotificationPrefs({
          in_app_enabled: Boolean(b?.in_app_enabled),
          on_item_added: Boolean(b?.on_item_added),
        });
      }
    }
  }

  async function addToUserTracking() {
    if (!canSubmit) return;
    try {
      const res = await fetch('/api/v1/me/tracked-items', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          url: form.url,
          country: form.country,
          zip: form.country === 'US' ? form.zip.trim() : '',
        }),
      });
      if (!res.ok) {
        const body = await res.json();
        setError(body?.error || 'Failed to add tracked item');
        return;
      }
      setError(null);
      await refreshUserPanel();
    } catch {
      setError('Network error while adding tracked item');
    }
  }

  async function refreshMonitorData() {
    const [mRes, nRes] = await Promise.allSettled([fetch("/api/monitor/list"), fetch("/api/monitor/notifications")]);

    if (mRes.status === "fulfilled") {
      if (mRes.value.ok) {
        const mb = await mRes.value.json();
        setMonitors(mb.monitors || []);
      }
    } else {
      console.error("monitor/list failed", mRes.reason);
    }

    if (nRes.status === "fulfilled") {
      if (nRes.value.ok) {
        const nb = await nRes.value.json();
        setNotifications(nb.notifications || []);
      }
    } else {
      console.error("monitor/notifications failed", nRes.reason);
    }
  }

  async function startMonitor() {
    if (!canSubmit) return;
    const payload = {
      url: form.url,
      country: form.country,
      zip: form.country === "US" ? form.zip.trim() : "",
      interval_seconds: intervalSec,
      max_runs: maxRuns,
    };
    try {
      const res = await fetch("/api/monitor/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (res.ok) {
        setError(null);
        await refreshMonitorData();
        return;
      }
      try {
        const body = await res.json();
        setError(body?.error || "Failed to start monitor");
      } catch {
        setError("Failed to start monitor");
      }
    } catch {
      setError("Network error while starting monitor");
    }
  }

  async function stopMonitor(id: string) {
    try {
      const res = await fetch(`/api/monitor/stop?id=${encodeURIComponent(id)}`, { method: "POST" });
      if (!res.ok) {
        try {
          const body = await res.json();
          setError(body?.error || "Failed to stop monitor");
        } catch {
          setError("Failed to stop monitor");
        }
        return;
      }
      setError(null);
      await refreshMonitorData();
    } catch {
      setError("Network error while stopping monitor");
    }
  }

  async function clearMonitors() {
    try {
      const res = await fetch("/api/monitor/clear", { method: "DELETE" });
      if (!res.ok) {
        try {
          const body = await res.json();
          setError(body?.error || "Failed to clear monitors");
        } catch {
          setError("Failed to clear monitors");
        }
        return;
      }
      setError(null);
      await refreshMonitorData();
    } catch {
      setError("Network error while clearing monitors");
    }
  }

  async function clearNotifications() {
    try {
      const res = await fetch("/api/monitor/notifications", { method: "DELETE" });
      if (!res.ok) {
        try {
          const body = await res.json();
          setError(body?.error || "Failed to reset notifications");
        } catch {
          setError("Failed to reset notifications");
        }
        return;
      }
      setError(null);
      await refreshMonitorData();
    } catch {
      setError("Network error while resetting notifications");
    }
  }


  async function markNotificationRead(id: string) {
    const res = await fetch('/api/v1/me/notifications/read', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ id }),
    });
    if (res.ok) await refreshUserPanel();
  }

  async function markAllNotificationsRead() {
    const res = await fetch('/api/v1/me/notifications/read', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ all: true }),
    });
    if (res.ok) await refreshUserPanel();
  }

  async function saveNotificationPrefs(next: NotificationPreferences) {
    const res = await fetch('/api/v1/me/notification-preferences', {
      method: 'PUT',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(next),
    });
    if (res.ok) {
      setNotificationPrefs(next);
      await refreshUserPanel();
    }
  }

  useEffect(() => {
    let active = true;
    const run = () => {
      fetch("/api/health")
        .then((r) => {
          if (!active) return;
          setHealth(r.ok ? "up" : "down");
        })
        .catch(() => {
          if (!active) return;
          setHealth("down");
        });
      refreshMonitorData();
      refreshUserPanel();
    };

    run();
    const id = setInterval(run, 5000);
    return () => {
      active = false;
      clearInterval(id);
    };
  }, []);

  const isFree = result?.free_shipping_country;
  const isGenericFree = result?.free_shipping;

  return (
    <main className="container">
      <section className="hero card">
        <h1>{t("app_title")}</h1>
        <p>{t("app_subtitle")}</p>
        <div className={`health ${health}`}>
          Backend: {health === "checking" ? "Checking..." : health === "up" ? "Online" : "Offline"}
        </div>
      </section>

      <section className="card">
        <form onSubmit={onSubmit} className="form">
          <label>
            Product URL
            <input
              type="url"
              value={form.url}
              onChange={(e) => setForm((x) => ({ ...x, url: e.target.value }))}
              placeholder="https://www.amazon.com/dp/..."
              required
            />
          </label>

          <div className="row">
            <label>
              Destination country
              <select
                value={form.country}
                onChange={(e) =>
                  setForm((x) => ({ ...x, country: e.target.value as "US" | "IL", zip: e.target.value === "US" ? x.zip : "" }))
                }
              >
                <option value="US">United States</option>
                <option value="IL">Israel</option>
              </select>
            </label>

            <label>
              ZIP (US only)
              <input
                type="text"
                value={form.zip}
                onChange={(e) => setForm((x) => ({ ...x, zip: e.target.value }))}
                placeholder="10013"
                disabled={form.country !== "US"}
              />
            </label>
          </div>

          <div className="row">
            <button disabled={!canSubmit || loading}>{loading ? t("loading") : t("check_button")}</button>
            <label>
              Monitor interval (seconds)
              <input type="number" min={5} max={3600} value={intervalSec} onChange={(e) => setIntervalSec(Number(e.target.value || 5))} />
            </label>
          </div>
          <div className="row">
            <label>
              Max runs (auto-stop)
              <input type="number" min={1} value={maxRuns} onChange={(e) => setMaxRuns(Number(e.target.value || 1))} />
            </label>
          </div>
          <div className="row">
            <button type="button" className="secondary" onClick={startMonitor} disabled={!canSubmit}>
              {t("monitor_button")}
            </button>
            <button type="button" className="secondary" onClick={addToUserTracking} disabled={!canSubmit}>
              {t("add_button")}
            </button>
          </div>
        </form>
      </section>

      {(error || result) && (
        <section className="card result">
          {error && <p className="error">Error: {error}</p>}

          {result && (
            <>
              <div className={`badge ${isFree ? "ok" : "no"}`}>
                {isFree ? "✅ Free shipping for destination" : "❌ Not free for destination"}
              </div>
              {!isFree && isGenericFree && (
                <p className="hint">Generic free-shipping text was found, but not confirmed for selected destination.</p>
              )}
              <ul>
                <li><strong>Country:</strong> {result.country}</li>
                <li><strong>Price (USD):</strong> {result.price_usd ? `$${result.price_usd.toFixed(2)}` : "-"}</li>
                <li><strong>free_shipping (generic):</strong> {String(result.free_shipping)}</li>
                <li><strong>free_shipping_country (strict):</strong> {String(result.free_shipping_country)}</li>
                <li><strong>Signal:</strong> {result.signal}</li>
                <li><strong>Method:</strong> {result.method}</li>
                <li><strong>Checked:</strong> {new Date(result.checked_at).toLocaleString()}</li>
              </ul>
              <details>
                <summary>Raw backend response</summary>
                <pre>{JSON.stringify(result, null, 2)}</pre>
              </details>
            </>
          )}
        </section>
      )}

      <section className="card">
        <h3>User panel (local test user)</h3>
        <p className="mutedText">Current user: <strong>{me?.name || 'loading...'}</strong></p>
        <div className="row split">
          <div>
            <h4>Tracked products</h4>
            <ul>
              {userItems.length === 0 && <li>No tracked products yet.</li>}
              {userItems.slice(0, 8).map((it) => (
                <li key={it.id}>
                  {it.country} · {it.free_shipping_country ? '✅' : '❌'} · {it.last_price_usd ? `$${it.last_price_usd.toFixed(2)}` : '-'}
                  <br />
                  <small>{it.url}</small>
                </li>
              ))}
            </ul>
          </div>
          <div>
            <h4>Alerts</h4>
            <ul>
              {userAlerts.length === 0 && <li>No user alerts yet.</li>}
              {userAlerts.slice(0, 8).map((a) => (
                <li key={a.id}>🔔 {new Date(a.created_at).toLocaleTimeString()} · {a.message}</li>
              ))}
            </ul>
          </div>
        </div>
      </section>


      <section className="card">
        <div className="row actionsRow">
          <h3>In-app notifications center</h3>
          <button className="secondary" onClick={markAllNotificationsRead}>Mark all read</button>
        </div>
        <div className="row split">
          <div>
            <h4>Preferences</h4>
            <label>
              <input
                type="checkbox"
                checked={notificationPrefs.in_app_enabled}
                onChange={(e) => saveNotificationPrefs({ ...notificationPrefs, in_app_enabled: e.target.checked })}
              /> Enable in-app notifications
            </label>
            <br />
            <label>
              <input
                type="checkbox"
                checked={notificationPrefs.on_item_added}
                disabled={!notificationPrefs.in_app_enabled}
                onChange={(e) => saveNotificationPrefs({ ...notificationPrefs, on_item_added: e.target.checked })}
              /> Notify when tracked item is added
            </label>
          </div>
          <div>
            <h4>Inbox ({inAppNotifications.filter((n) => !n.read).length} unread)</h4>
            <ul>
              {inAppNotifications.length === 0 && <li>No notifications yet.</li>}
              {inAppNotifications.slice(0, 10).map((n) => (
                <li key={n.id}>
                  {n.read ? '✅' : '🟡'} {new Date(n.created_at).toLocaleTimeString()} · <strong>{n.title}</strong> — {n.message}
                  {!n.read && (
                    <button className="secondary" onClick={() => markNotificationRead(n.id)} style={{ marginLeft: 8 }}>
                      mark read
                    </button>
                  )}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      <section className="card">
        <div className="row actionsRow">
          <h3>Monitoring test (US/IL + cron + UI notify)</h3>
          <button className="secondary" onClick={clearMonitors}>Clear old monitors</button>
        </div>
        <ul>
          {monitors.length === 0 && <li>No active monitors yet.</li>}
          {monitors.map((m) => (
            <li key={m.id} className="monitorItem">
              <div>
                <strong>{m.country}</strong> · every {m.interval_seconds}s · runs {m.runs_done ?? 0}/{m.max_runs ?? 0} · {m.running ? "running" : "stopped"}
                <br />
                <small>{m.url}</small>
                <br />
                <small>
                  last checked: {m.last_checked_at ? new Date(m.last_checked_at).toLocaleTimeString() : "-"} · status: {m.last_status == null ? "-" : m.last_status ? "✅" : "❌"} · price: {m.last_price_usd ? `$${m.last_price_usd.toFixed(2)}` : "-"}
                </small>
                {m.last_signal && <small> · signal: {m.last_signal}</small>}
              </div>
              <div>
                {m.running && (
                  <button className="secondary" onClick={() => stopMonitor(m.id)}>
                    Stop
                  </button>
                )}
                {m.history?.length > 0 && (
                  <details>
                    <summary>History</summary>
                    <ul>
                      {m.history.slice(0, 5).map((h, idx) => (
                        <li key={idx}>
                          {new Date(h.at).toLocaleTimeString()} · price {h.price_usd ? `$${h.price_usd.toFixed(2)}` : "-"} · strict {String(h.free_shipping_country)}
                        </li>
                      ))}
                    </ul>
                  </details>
                )}
              </div>
            </li>
          ))}
        </ul>
      </section>

      <section className="card">
        <div className="row actionsRow">
          <h3>UI notifications (test mode)</h3>
          <button className="secondary" onClick={clearNotifications}>Reset notifications</button>
        </div>
        <ul>
          {notifications.length === 0 && <li>No notifications yet.</li>}
          {notifications.slice(0, 10).map((n, i) => (
            <li key={`${n.monitor_id}-${i}`}>
              🔔 {new Date(n.at).toLocaleTimeString()} · {n.message}
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}
