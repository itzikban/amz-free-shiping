"use client";

import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";

type Lang = "en" | "he";

const dictionaries = {
  en: {
    nav_checker: "Checker",
    nav_watchlist: "Watchlist",
    nav_alerts: "Alerts",
    language: "Language",
    app_title: "AMZ Free Shipping Checker",
    app_subtitle: "Country-aware shipping checker powered by your backend.",
    home_discover_title: "Discover Global Shipping",
    home_url_placeholder: "Paste Amazon URL here...",
    home_country_il: "Israel (IL)",
    home_country_us: "United States (US)",
    home_zip_placeholder: "ZIP",
    home_analyze: "Analyze",
    home_target_analysis: "Target Analysis",
    home_alert_cta_disabled: "Alert feature coming soon",
    home_flow_hint: "While you wait, we found alternatives with free shipping",
    rec_title_1: "Sony WH-1000XM5 (International Version)",
    rec_title_2: "Sony WH-1000XM5 Wireless - Silver",
    rec_title_3: "Bose QuietComfort Ultra - Black",
    rec_tag_free_shipping: "FREE SHIPPING",
    rec_tag_best_match: "BEST MATCH",
    check_button: "Check free shipping",
    add_button: "Add product to watchlist",
    monitor_button: "Start monitor",
    loading: "Loading…",
    watchlist_title: "Watchlist",
    alerts_title: "Alerts Center",
    details_title: "Tracked Product Details",
    back_watchlist: "Back to watchlist",
    alerts_subtitle: "In-app alerts for shipping and price changes.",
    alerts_filter_all: "All",
    alerts_filter_free: "Free shipping",
    alerts_filter_price: "Price changes",
    alerts_empty_filtered: "No alerts match current filter.",
    products_subtitle: "Tracked products with destination-specific shipping verdict.",
    products_total_tracked: "Total tracked",
    products_free_shipping: "Free shipping",
    products_not_free: "Not free",
    products_fallback_notice: "Backend unavailable — showing local fallback sample data.",
    products_heading: "Tracked products",
    products_add_from_checker: "+ Add from checker",
    products_empty: "No tracked products yet.",
    products_last_checked: "Last checked:",
    products_view_details: "View details",
    products_search_placeholder: "Search URL / country",
    products_no_results: "No matching products for this search.",
    products_status_free: "✅ Free",
    products_status_not_free: "❌ Not free",
    details_not_found: "Tracked product not found.",
    details_id: "ID:",
    details_url: "URL:",
    details_country: "Country:",
    details_zip: "ZIP:",
    details_price: "Price:",
    details_free_shipping: "Free shipping:",
    details_signal: "Signal:",
    details_method: "Method:",
    details_last_checked: "Last checked:",
    backend_status: "Backend",
    checking: "Checking",
    online: "Online",
    offline: "Offline",
    quick_check: "Quick check",
    product_url: "Product URL",
    destination_country: "Destination country",
    zip_us: "ZIP (US only)",
    err_add_watchlist: "Failed to add to watchlist",
    err_network: "Network error",
    free_for_destination: "✅ Free shipping for destination",
    not_free_for_destination: "❌ Not free for destination",
    country_label: "Country",
    price_label: "Price (USD)",
    signal_label: "Signal",
    method_label: "Method",
    view_all: "View all",
    empty_watchlist: "No tracked products yet.",
    empty_alerts: "No alerts yet.",
    common_dash: "-",
    common_yes: "Yes",
    common_no: "No",
  },
  he: {
    nav_checker: "בודק",
    nav_watchlist: "רשימת מעקב",
    nav_alerts: "התראות",
    language: "שפה",
    app_title: "בודק משלוח חינם באמזון",
    app_subtitle: "בדיקת משלוח לפי מדינה, עם מעקב והתראות.",
    home_discover_title: "גלו משלוחים גלובליים",
    home_url_placeholder: "הדביקו כאן קישור אמזון...",
    home_country_il: "ישראל (IL)",
    home_country_us: "ארצות הברית (US)",
    home_zip_placeholder: "מיקוד",
    home_analyze: "נתח",
    home_target_analysis: "ניתוח יעד",
    home_alert_cta_disabled: "תכונת התראות תתווסף בקרוב",
    home_flow_hint: "בזמן ההמתנה, מצאנו חלופות עם משלוח חינם",
    rec_title_1: "Sony WH-1000XM5 (גרסה בינלאומית)",
    rec_title_2: "Sony WH-1000XM5 אלחוטי - כסף",
    rec_title_3: "Bose QuietComfort Ultra - שחור",
    rec_tag_free_shipping: "משלוח חינם",
    rec_tag_best_match: "ההתאמה הטובה ביותר",
    check_button: "בדוק משלוח חינם",
    add_button: "הוסף למעקב",
    monitor_button: "התחל ניטור",
    loading: "טוען…",
    watchlist_title: "רשימת מעקב",
    alerts_title: "מרכז התראות",
    details_title: "פרטי מוצר במעקב",
    back_watchlist: "חזרה לרשימת המעקב",
    alerts_subtitle: "התראות בתוך האפליקציה על משלוח ושינויי מחיר.",
    alerts_filter_all: "הכל",
    alerts_filter_free: "משלוח חינם",
    alerts_filter_price: "שינויי מחיר",
    alerts_empty_filtered: "אין התראות שתואמות לסינון הנוכחי.",
    products_subtitle: "מוצרים במעקב עם סטטוס משלוח לפי יעד.",
    products_total_tracked: "סה״כ במעקב",
    products_free_shipping: "משלוח חינם",
    products_not_free: "לא חינם",
    products_fallback_notice: "השרת לא זמין — מוצגים נתוני דוגמה מקומיים.",
    products_heading: "מוצרים במעקב",
    products_add_from_checker: "+ הוסף מהבודק",
    products_empty: "עדיין אין מוצרים במעקב.",
    products_last_checked: "נבדק לאחרונה:",
    products_view_details: "צפה בפרטים",
    products_search_placeholder: "חיפוש לפי קישור / מדינה",
    products_no_results: "לא נמצאו מוצרים שמתאימים לחיפוש.",
    products_status_free: "✅ חינם",
    products_status_not_free: "❌ לא חינם",
    details_not_found: "המוצר במעקב לא נמצא.",
    details_id: "מזהה:",
    details_url: "קישור:",
    details_country: "מדינה:",
    details_zip: "מיקוד:",
    details_price: "מחיר:",
    details_free_shipping: "משלוח חינם:",
    details_signal: "איתות:",
    details_method: "שיטה:",
    details_last_checked: "נבדק לאחרונה:",
    backend_status: "שרת",
    checking: "בודק",
    online: "מחובר",
    offline: "מנותק",
    quick_check: "בדיקה מהירה",
    product_url: "קישור מוצר",
    destination_country: "מדינת יעד",
    zip_us: "מיקוד (ארה״ב בלבד)",
    err_add_watchlist: "נכשל בהוספה לרשימת המעקב",
    err_network: "שגיאת רשת",
    free_for_destination: "✅ משלוח חינם ליעד",
    not_free_for_destination: "❌ אין משלוח חינם ליעד",
    country_label: "מדינה",
    price_label: "מחיר (USD)",
    signal_label: "איתות",
    method_label: "שיטה",
    view_all: "צפה בהכל",
    empty_watchlist: "עדיין אין מוצרים במעקב.",
    empty_alerts: "עדיין אין התראות.",
    common_dash: "-",
    common_yes: "כן",
    common_no: "לא",
  },
} as const;

type TranslationKey = keyof typeof dictionaries.en;

type I18nContextType = {
  lang: Lang;
  dir: "ltr" | "rtl";
  setLang: (lang: Lang) => void;
  t: (key: TranslationKey) => string;
  formatDate: (value: Date | number | string, options?: Intl.DateTimeFormatOptions) => string;
};

const I18nContext = createContext<I18nContextType | null>(null);

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>("en");

  useEffect(() => {
    const next: Lang = localStorage.getItem("ui_lang") === "he" ? "he" : "en";
    setLangState(next);
  }, []);

  useEffect(() => {
    const dir = lang === "he" ? "rtl" : "ltr";
    document.documentElement.lang = lang;
    document.documentElement.dir = dir;
  }, [lang]);

  const setLang = useCallback((next: Lang) => {
    setLangState(next);
    localStorage.setItem("ui_lang", next);
  }, []);

  const value = useMemo<I18nContextType>(() => {
    const dir = lang === "he" ? "rtl" : "ltr";
    const dict = dictionaries[lang];
    const locale = lang === "he" ? "he-IL" : "en-US";
    return {
      lang,
      dir,
      setLang,
      t: (key: TranslationKey) => dict[key],
      formatDate: (value: Date | number | string, options?: Intl.DateTimeFormatOptions) => {
        const date = value instanceof Date ? value : new Date(value);
        if (Number.isNaN(date.getTime())) return String(value ?? "");
        return new Intl.DateTimeFormat(locale, options).format(date);
      },
    };
  }, [lang, setLang]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}
