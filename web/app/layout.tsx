import type { Metadata } from "next";
import { Header } from "@/components/Header";
import "./globals.css";

export const metadata: Metadata = {
  title: "TokoHerbal",
  description: "Toko produk herbal: jamu, madu, dan rempah pilihan.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id">
      <body>
        <Header />
        <main className="container">{children}</main>
        <footer className="site-footer">Proyek portofolio · Go, PostgreSQL, Redis, Next.js</footer>
      </body>
    </html>
  );
}
