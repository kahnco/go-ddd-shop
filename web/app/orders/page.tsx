import Link from "next/link";
import { getSession } from "@/lib/session";
import { getMyOrders, won, statusLabel } from "@/lib/api";

const statusColor: Record<string, string> = {
  PLACED: "text-neutral-300 bg-white/5",
  CONFIRMED: "text-blue-300 bg-blue-500/10",
  SHIPPED: "text-emerald-300 bg-emerald-500/10",
  CANCELLED: "text-red-300 bg-red-500/10",
  RETURN_REQUESTED: "text-amber-300 bg-amber-500/10",
  REFUNDED: "text-neutral-400 bg-white/5",
};

export default async function OrdersPage() {
  const session = await getSession();

  if (!session) {
    return (
      <div className="mx-auto max-w-sm text-center text-neutral-400">
        <h1 className="mb-4 text-2xl font-bold text-white">내 주문</h1>
        <p>주문 내역을 보려면 로그인이 필요합니다.</p>
        <Link
          href="/login"
          className="mt-6 inline-block rounded-lg bg-blue-600 px-5 py-2 font-medium text-white hover:bg-blue-500"
        >
          로그인하기
        </Link>
      </div>
    );
  }

  const orders = await getMyOrders(session.customerId);

  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold">내 주문</h1>
      <p className="mb-8 text-sm text-neutral-500">
        읽기 모델(CQRS)이 주문 이벤트로 만든 조회 전용 뷰입니다.
      </p>

      {orders.length === 0 ? (
        <div className="rounded-xl border border-white/10 bg-white/[0.03] p-10 text-center text-neutral-500">
          아직 주문이 없습니다.
          <br />
          <Link href="/" className="mt-3 inline-block text-blue-400 hover:underline">
            쇼핑하러 가기 →
          </Link>
        </div>
      ) : (
        <div className="space-y-3">
          {orders.map((o) => (
            <div
              key={o.order_id}
              className="flex items-center justify-between rounded-xl border border-white/10 bg-white/[0.03] px-5 py-4"
            >
              <div>
                <p className="font-mono text-sm text-neutral-300">{o.order_id}</p>
                <p className="mt-1 text-xs text-neutral-500">
                  상품 {o.items?.length ?? 0}종
                  {o.channel ? ` · ${o.channel}` : ""}
                </p>
              </div>
              <div className="flex items-center gap-4">
                <span className="text-sm text-neutral-300">{won(o.total)}</span>
                <span
                  className={`rounded-full px-3 py-1 text-xs font-medium ${statusColor[o.status] ?? "bg-white/5 text-neutral-300"}`}
                >
                  {statusLabel(o.status)}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
