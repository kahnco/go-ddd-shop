import Link from "next/link";
import { getSession } from "@/lib/session";
import { getMyOrders } from "@/lib/api";
import LiveOrders from "./LiveOrders";

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
        읽기 모델(CQRS)이 주문 이벤트로 만든 조회 전용 뷰입니다. 사가가 진행 중이면 실시간으로
        갱신됩니다.
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
        <LiveOrders initial={orders} />
      )}
    </div>
  );
}
