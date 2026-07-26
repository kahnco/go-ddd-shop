import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Go DDD Shop",
  description: "이벤트 기반 쇼핑몰 — Next.js 프런트엔드",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ko">
      <body className="min-h-screen">
        <header className="sticky top-0 z-10 border-b border-white/10 bg-neutral-950/80 backdrop-blur">
          <nav className="mx-auto flex max-w-5xl items-center justify-between px-5 py-4">
            <Link href="/" className="text-lg font-bold tracking-tight">
              🛒 Go DDD Shop
            </Link>
            <div className="flex items-center gap-5 text-sm text-neutral-400">
              <Link href="/" className="transition-colors hover:text-white">
                상품
              </Link>
              <Link href="/cart" className="transition-colors hover:text-white">
                장바구니
              </Link>
              <Link href="/orders" className="transition-colors hover:text-white">
                내 주문
              </Link>
              <Link href="/login" className="transition-colors hover:text-white">
                로그인
              </Link>
            </div>
          </nav>
        </header>
        <main className="mx-auto max-w-5xl px-5 py-10">{children}</main>
        <footer className="mx-auto max-w-5xl px-5 py-10 text-xs text-neutral-600">
          Go DDD Shop · Next.js(App Router) + Go(DDD/EDD) · docker-compose 풀스택
        </footer>
      </body>
    </html>
  );
}
