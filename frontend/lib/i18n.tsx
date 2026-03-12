"use client";

import React, { createContext, useContext, useEffect, useMemo, useState } from "react";

type Lang = "en" | "he";

type Dict = Record<string, string>;

const dictionaries: Record<Lang, Dict> = {
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
  },
};

type I18nContextType = {
  lang: Lang;
  dir: "ltr" | "rtl";
  setLang: (lang: Lang) => void;
  t: (key: string) => string;
};

const I18nContext = createContext<I18nContextType | null>(null);

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>("en");

  useEffect(() => {
    const saved = (localStorage.getItem("ui_lang") || "en") as Lang;
    const next: Lang = saved === "he" ? "he" : "en";
    setLangState(next);
  }, []);

  useEffect(() => {
    const dir = lang === "he" ? "rtl" : "ltr";
    document.documentElement.lang = lang;
    document.documentElement.dir = dir;
  }, [lang]);

  const setLang = (next: Lang) => {
    setLangState(next);
    localStorage.setItem("ui_lang", next);
  };

  const value = useMemo<I18nContextType>(() => {
    const dir = lang === "he" ? "rtl" : "ltr";
    const dict = dictionaries[lang];
    return {
      lang,
      dir,
      setLang,
      t: (key: string) => dict[key] || key,
    };
  }, [lang]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}
