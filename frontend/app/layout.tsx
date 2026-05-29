import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Go URL Shortener",
  description: "URL Shortener 'phitsanudev' portfolio built with Go, PostgreSQL, Redis and Next.js",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="th">
      <body>{children}</body>
    </html>
  );
}
