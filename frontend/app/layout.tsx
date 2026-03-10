import "./globals.css";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AMZ Free Shipping Checker",
  description: "Country-aware Amazon free-shipping checker",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
