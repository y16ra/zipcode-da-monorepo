import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "郵便番号検索",
  description: "郵便番号・デジタルアドレスAPI 検索デモ",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ja">
      <body>{children}</body>
    </html>
  );
}
