import type { Metadata } from "next";
import { Space_Grotesk, Sora } from "next/font/google";
import "./globals.css";

const display = Space_Grotesk({
  subsets: ["latin"],
  variable: "--font-display"
});

const bodyFont = Sora({
  subsets: ["latin"],
  variable: "--font-body"
});

export const metadata: Metadata = {
  title: "Турниры",
  description: "Сетки турниров Dota 2 / CS2"
};

export default function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ru" data-theme="dark" data-game="neutral">
      <body className={`min-h-screen antialiased ${display.variable} ${bodyFont.variable}`}>
        {children}
      </body>
    </html>
  );
}
