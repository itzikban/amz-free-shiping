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
};

const I18nContext = createContext<I18nContextType | null>(null);

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>(() => {
    if (typeof window === "undefined") return "en";
    return localStorage.getItem("ui_lang") === "he" ? "he" : "en";
  });

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
    return {
      lang,
      dir,
      setLang,
      t: (key: TranslationKey) => dict[key],
    };
  }, [lang, setLang]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}
