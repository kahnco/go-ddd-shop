import { NextResponse } from "next/server";
import { checkout } from "@/lib/api";
import { apiError, unauthorized } from "@/lib/http";
import { getSession } from "@/lib/session";

// 결제 = 장바구니 → 주문. 장바구니 서비스가 토큰을 다시 주문 서비스로 전파해
// 사가(재고→결제→배송)가 돈다. 여기선 그 시작만 호출한다.
export async function POST() {
  const session = await getSession();
  if (!session) return unauthorized();
  try {
    const orderId = await checkout(session.customerId, session.token);
    return NextResponse.json({ ok: true, order_id: orderId });
  } catch (e) {
    return apiError(e);
  }
}
