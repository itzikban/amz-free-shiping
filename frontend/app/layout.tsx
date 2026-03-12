import "./globals.css";
import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "AMZ Free Shipping Checker",
  description: "Country-aware Amazon free-shipping checker",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <nav className="topNav" aria-label="Main navigation">
          <Link href="/">Checker</Link>
          <Link href="/products">Watchlist</Link>
          <Link href="/alerts">Alerts Center</Link>
        </nav>
        {children}
      </body>
    </html>
  );
}
