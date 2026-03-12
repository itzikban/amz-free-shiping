import "./globals.css";
import type { Metadata } from "next";
import TopNav from "@/components/TopNav";
import NeuralBackdrop from "@/components/NeuralBackdrop";
import { I18nProvider } from "@/lib/i18n";

export const metadata: Metadata = {
  title: "AMZ Free Shipping Checker",
  description: "Country-aware Amazon free-shipping checker",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <I18nProvider>
          <NeuralBackdrop />
          <TopNav />
          {children}
        </I18nProvider>
      </body>
    </html>
  );
}
