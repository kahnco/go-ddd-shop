import Link from "next/link";
import { getSession } from "@/lib/session";
import { getStats, listProducts, won, statusLabel } from "@/lib/api";
import AddProductForm from "./AddProductForm";
import PriceEditor from "./PriceEditor";

export default async function AdminPage() {
  const session = await getSession();

  // 관리자 게이트 — 서버에서 role 로 막는다(비관리자는 화면 자체를 못 본다).
  if (!session || session.role !== "admin") {
    return (
      <div className="mx-auto max-w-sm text-center text-neutral-400">
        <h1 className="mb-4 text-2xl font-bold text-white">관리자</h1>
        <p>관리자 전용 페이지입니다.</p>
        <Link
          href="/"
          className="mt-6 inline-block rounded-lg bg-blue-600 px-5 py-2 font-medium text-white hover:bg-blue-500"
        >
          홈으로
        </Link>
      </div>
    );
  }

  const [stats, products] = await Promise.all([getStats(), listProducts()]);
  const maxCount = Math.max(1, ...Object.values(stats.counts));
  const statusOrder = ["PLACED", "CONFIRMED", "SHIPPED", "CANCELLED", "RETURN_REQUESTED", "REFUNDED"];
  const rows = statusOrder.filter((s) => stats.counts[s]).map((s) => ({ s, n: stats.counts[s] }));

  return (
    <div>
      <h1 className="mb-8 text-2xl font-bold">관리자 대시보드</h1>

      {/* 집계(읽기 모델) */}
      <section className="mb-12">
        <h2 className="mb-4 text-sm uppercase tracking-widest text-neutral-500">주문 통계</h2>
        <div className="mb-6 grid grid-cols-2 gap-4">
          <div className="rounded-xl border border-white/10 bg-white/[0.03] p-5">
            <p className="text-xs text-neutral-500">총 주문</p>
            <p className="mt-1 text-2xl font-bold">{stats.order_count.toLocaleString()}</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/[0.03] p-5">
            <p className="text-xs text-neutral-500">총 매출(취소 제외)</p>
            <p className="mt-1 text-2xl font-bold text-emerald-300">{won(stats.total_revenue)}</p>
          </div>
        </div>
        <div className="space-y-2 rounded-xl border border-white/10 bg-white/[0.03] p-5">
          {rows.length === 0 ? (
            <p className="text-sm text-neutral-500">아직 주문이 없습니다.</p>
          ) : (
            rows.map(({ s, n }) => (
              <div key={s} className="flex items-center gap-3 text-sm">
                <span className="w-20 shrink-0 text-neutral-400">{statusLabel(s)}</span>
                <div className="h-3 flex-1 overflow-hidden rounded-full bg-white/5">
                  <div
                    className="h-full rounded-full bg-blue-500/60"
                    style={{ width: `${(n / maxCount) * 100}%` }}
                  />
                </div>
                <span className="w-8 shrink-0 text-right text-neutral-300">{n}</span>
              </div>
            ))
          )}
        </div>
      </section>

      {/* 상품 관리(카탈로그 write) */}
      <section>
        <h2 className="mb-4 text-sm uppercase tracking-widest text-neutral-500">상품 관리</h2>
        <div className="mb-6 rounded-xl border border-white/10 bg-white/[0.03] p-5">
          <AddProductForm />
        </div>
        <div className="divide-y divide-white/5 rounded-xl border border-white/10 bg-white/[0.03]">
          {products.map((p) => (
            <div key={p.product_id} className="flex items-center justify-between px-5 py-3">
              <div>
                <p className="font-medium">{p.name}</p>
                <p className="font-mono text-xs text-neutral-500">{p.product_id}</p>
              </div>
              <PriceEditor productId={p.product_id} price={p.price} />
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
