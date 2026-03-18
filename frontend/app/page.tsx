"use client";

import { useMemo, useState } from "react";
import type { CheckResponse } from "@/lib/types";
import { useI18n } from "@/lib/i18n";

type FormState = {
  url: string;
  country: "US" | "IL" | "NL";
  zip: string;
};

type FormErrors = {
  url?: string;
  zip?: string;
};

const SAMPLE = "https://www.amazon.com/dp/B0DHCZBKW7";

export default function HomePage() {
  const { t, formatDate } = useI18n();
  const [form, setForm] = useState<FormState>({ url: SAMPLE, country: "IL", zip: "" });
  const [errors, setErrors] = useState<FormErrors>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<CheckResponse | null>(null);
  const [submittedForm, setSubmittedForm] = useState<FormState | null>(null);
  const [addingWatchlist, setAddingWatchlist] = useState(false);
  const [watchlistAdded, setWatchlistAdded] = useState(false);

  const canSubmit = useMemo(() => {
    return Object.keys(validateForm(form)).length === 0;
  }, [form]);

  function validateForm(value: FormState): FormErrors {
    const next: FormErrors = {};
    let parsed: URL | null = null;

    try {
      parsed = new URL(value.url.trim());
      if (!["http:", "https:"].includes(parsed.protocol)) {
        next.url = t("home_validation_url_protocol");
      }
      if (!parsed.hostname.includes("amazon.")) {
        next.url = t("home_validation_url_amazon");
      }
    } catch {
      next.url = t("home_validation_url_invalid");
    }

    if (value.country === "US") {
      const zip = value.zip.trim();
      if (!zip) {
        next.zip = t("home_validation_zip_required");
      } else if (!/^\d{5}(?:-\d{4})?$/.test(zip)) {
        next.zip = t("home_validation_zip_format");
      }
    }

    return next;
  }

  function mapBackendError(raw: string): string {
    const lower = raw.toLowerCase();
    if (lower.includes("missing url")) return t("home_error_missing_url");
    if (lower.includes("upstream status") || lower.includes("status: 4") || lower.includes("status: 5")) return t("home_error_upstream");
    if (lower.includes("timeout") || lower.includes("network") || lower.includes("fetch")) return t("home_error_network");
    return t("home_error_generic");
  }

  function getResultBadge(res: CheckResponse): { label: string; cls: "ok" | "no" | "neutral" } {
    if (loading) return { label: t("home_badge_checking"), cls: "neutral" };
    if (res.free_shipping_country) return { label: t("home_badge_free_shipping"), cls: "ok" };
    if (res.free_shipping && !res.free_shipping_country) return { label: t("home_badge_paid_shipping"), cls: "no" };
    return { label: t("home_badge_unavailable"), cls: "neutral" };
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    const nextErrors = validateForm(form);
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;

    const submitted: FormState = {
      url: form.url.trim(),
      country: form.country,
      zip: form.zip.trim(),
    };

    setLoading(true);
    setError(null);
    setResult(null);
    setSubmittedForm(null);
    setWatchlistAdded(false);

    try {
      const params = new URLSearchParams({ url: submitted.url, country: submitted.country });
      if (submitted.country === "US" && submitted.zip) params.set("zip", submitted.zip);
      const res = await fetch(`/api/check?${params.toString()}`);
      const body = await res.json();
      if (!res.ok) throw new Error(body?.error || "Request failed");
      setResult(body);
      setSubmittedForm(submitted);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Unknown error";
      setError(mapBackendError(msg));
    } finally {
      setLoading(false);
    }
  }

  async function addToWatchlist() {
    if (!submittedForm || addingWatchlist) return;
    setError(null);
    setAddingWatchlist(true);
    setWatchlistAdded(false);
    try {
      const res = await fetch("/api/v1/me/tracked-items", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          url: submittedForm.url,
          country: submittedForm.country,
          zip: submittedForm.country === "US" ? submittedForm.zip : "",
        }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body?.error || t("err_add_watchlist"));
      }
      setWatchlistAdded(true);
    } catch {
      setError(t("err_network"));
      setWatchlistAdded(false);
    } finally {
      setAddingWatchlist(false);
    }
  }

  const badge = result ? getResultBadge(result) : null;

  return (
    <main className="container nexusHome">
      <section className="heroCenter animate-flow-1">
        <h1>{t("home_discover_title")}</h1>
        <p>{t("app_subtitle")}</p>

        <form className="search-container" onSubmit={onSubmit} noValidate>
          <div className="search-row grow">
            <label className="sr-only" htmlFor="product-url">{t("product_url")}</label>
            <input
              id="product-url"
              className="clean-input"
              type="url"
              value={form.url}
              onChange={(e) => setForm((x) => ({ ...x, url: e.target.value }))}
              placeholder={t("home_url_placeholder")}
              required
            />
          </div>
          {errors.url && <p className="error">{errors.url}</p>}

          <div className="search-row fixed">
            <label className="sr-only" htmlFor="destination-country">{t("destination_country")}</label>
            <select
              id="destination-country"
              className="clean-input"
              value={form.country}
              onChange={(e) => setForm((x) => ({ ...x, country: e.target.value as FormState["country"] }))}
            >
              <option value="IL">{t("home_country_il")}</option>
              <option value="US">{t("home_country_us")}</option>
              <option value="NL">{t("home_country_nl")}</option>
            </select>
          </div>

          {form.country === "US" && (
            <>
              <div className="search-row fixed">
                <label className="sr-only" htmlFor="destination-zip">{t("zip_us")}</label>
                <input
                  id="destination-zip"
                  className="clean-input"
                  type="text"
                  value={form.zip}
                  onChange={(e) => setForm((x) => ({ ...x, zip: e.target.value }))}
                  placeholder={t("home_zip_placeholder")}
                />
              </div>
              {errors.zip && <p className="error">{errors.zip}</p>}
            </>
          )}

          <button className="analyzeBtn" disabled={!canSubmit || loading}>
            {loading ? t("loading") : t("home_analyze")}
          </button>
        </form>
      </section>

      {(loading || result || error) && (
        <section className="fluid-card targetCard animate-flow-2">
          {result?.image_url ? (
            <img className="targetImage" src={result.image_url} alt={result.title || "Product"} />
          ) : (
            <div className="targetImage img-placeholder-main" />
          )}
          <div className="targetBody">
            <h3>{result?.title || t("home_target_analysis")}</h3>
            {error && <p className="error">{error}</p>}

            {loading && (
              <div className="chipRow">
                <span className="signalPill neutral">{t("home_badge_checking")}</span>
              </div>
            )}

            {result && (
              <>
                <p className="mutedText">{result.url}</p>
                <div className="targetMeta">
                  <span className="priceTag">{result.price_usd != null ? `$${result.price_usd.toFixed(2)}` : t("home_value_na")}</span>
                  {badge && <span className={`signalPill ${badge.cls}`}>{badge.label}</span>}
                </div>
                <div className="chipRow">
                  <span className="signalPill neutral">{`${t("country_label")}: ${result.country}`}</span>
                  <span className="signalPill neutral">{`${t("details_last_checked")} ${formatDate(result.checked_at)}`}</span>
                </div>
                <div className="chipRow">
                  <span className="signalPill neutral">{`${t("home_availability")}: ${result.signal === "captcha_detected" ? t("home_availability_unavailable") : t("home_availability_available")}`}</span>
                  <span className="signalPill neutral">{`${t("home_shipping_cost")}: ${result.free_shipping_country ? t("home_shipping_cost_free") : t("home_shipping_cost_unknown")}`}</span>
                </div>
              </>
            )}

            {result && submittedForm && (
              <div className="targetActions">
                <button className="secondary" type="button" onClick={addToWatchlist} disabled={addingWatchlist || watchlistAdded}>
                  {addingWatchlist ? t("loading") : watchlistAdded ? t("watchlist_added") : t("add_button")}
                </button>
              </div>
            )}
          </div>
        </section>
      )}
    </main>
  );
}
