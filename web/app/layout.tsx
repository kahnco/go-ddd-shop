import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";
import { getSession } from "@/lib/session";
import { getCart } from "@/lib/api";
import LogoutButton from "@/app/components/LogoutButton";

export const metadata: Metadata = {
  title: "Go DDD Shop",
  description: "이벤트 기반 쇼핑몰 — Next.js 프런트엔드",
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  // 로그인 상태면 장바구니 개수를 배지로 보여준다.
  const session = await getSession();
  let cartCount = 0;
  if (session) {
    const cart = await getCart(session.customerId, session.token);
    cartCount = cart.items.reduce((n, it) => n + it.quantity, 0);
  }

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
              <Link href="/cart" className="relative transition-colors hover:text-white">
                장바구니
                {cartCount > 0 && (
                  <span className="absolute -right-4 -top-2 rounded-full bg-blue-600 px-1.5 py-0.5 text-[10px] font-bold text-white">
                    {cartCount}
                  </span>
                )}
              </Link>
              {session ? (
                <>
                  <Link href="/orders" className="transition-colors hover:text-white">
                    내 주문
                  </Link>
                  {session.role === "admin" && (
                    <Link href="/admin" className="text-amber-300 transition-colors hover:text-amber-200">
                      관리자
                    </Link>
                  )}
                  <LogoutButton />
                </>
              ) : (
                <Link href="/login" className="transition-colors hover:text-white">
                  로그인
                </Link>
              )}
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
