"use client";

import Link from "next/link";
import { useI18n } from "@/lib/i18n";

export default function TopNav() {
  const { lang, setLang, t } = useI18n();

  return (
    <nav className="topNav" aria-label="Main navigation">
      <div className="topNavLinks">
        <Link href="/">{t("nav_checker")}</Link>
        <Link href="/products">{t("nav_watchlist")}</Link>
        <Link href="/alerts">{t("nav_alerts")}</Link>
      </div>
      <label className="langSelect" htmlFor="lang-select">
        {t("language")}
        <select id="lang-select" value={lang} onChange={(e) => setLang(e.target.value as "en" | "he") }>
          <option value="en">English</option>
          <option value="he">עברית</option>
        </select>
      </label>
    </nav>
  );
}
