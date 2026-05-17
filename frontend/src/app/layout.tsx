import type { Metadata, Viewport } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Confirmaciones",
  description: "Revisión de confirmaciones de citas",
  manifest: "/manifest.json",
  appleWebApp: {
    capable: true,
    statusBarStyle: "black-translucent",
    title: "Confirmaciones",
  },
};

export const viewport: Viewport = {
  themeColor: "#0b141a",
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  userScalable: false,
  maximumScale: 1,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="es">
      <head>
        <link rel="apple-touch-icon" href="/icon-192.png" />
      </head>
      <body className="min-h-dvh bg-bg text-[#e9edef] flex flex-col max-w-[560px] mx-auto">
        {children}
      </body>
    </html>
  );
}
