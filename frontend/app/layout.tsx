import "./globals.css";
import type { Metadata } from "next";
import { cookies } from "next/headers";
import TopNav from "@/components/TopNav";
import NeuralBackdrop from "@/components/NeuralBackdrop";
import { I18nProvider } from "@/lib/i18n";
import type { Lang } from "@/lib/i18n";

export const metadata: Metadata = {
  title: "AMZ Free Shipping Checker",
  description: "Country-aware Amazon free-shipping checker",
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const cookieStore = await cookies();
  const langCookie = cookieStore.get("ui_lang")?.value;
  const initialLang: Lang = langCookie === "he" ? "he" : "en";

  return (
    <html lang={initialLang} dir={initialLang === "he" ? "rtl" : "ltr"} suppressHydrationWarning>
      <body>
        <I18nProvider initialLang={initialLang}>
          <NeuralBackdrop />
          <TopNav />
          {children}
        </I18nProvider>
      </body>
    </html>
  );
}
