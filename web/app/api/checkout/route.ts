import { NextResponse } from "next/server";
import { services } from "@/lib/api";
import { getSession } from "@/lib/session";

// 결제 = 장바구니 → 주문. 장바구니 서비스가 토큰을 다시 주문 서비스로 전파해
// 사가(재고→결제→배송)가 돈다. 여기선 그 시작만 호출한다.
export async function POST() {
  const session = await getSession();
  if (!session) {
    return NextResponse.json({ error: "로그인이 필요합니다" }, { status: 401 });
  }
  const res = await fetch(
    `${services.cart}/carts/${encodeURIComponent(session.customerId)}/checkout`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${session.token}`,
        "X-Client-Channel": "web",
      },
    },
  );
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    return NextResponse.json({ error: body.error ?? "결제 실패" }, { status: res.status });
  }
  return NextResponse.json({ ok: true, order_id: body.order_id });
}
