import { NextResponse } from "next/server";
import { getMyOrders } from "@/lib/api";
import { unauthorized } from "@/lib/http";
import { getSession } from "@/lib/session";

// 실시간 폴링용 — 내 주문 목록(읽기 모델)을 JSON 으로. 브라우저가 주기적으로 당긴다.
export async function GET() {
  const session = await getSession();
  if (!session) return unauthorized();
  const orders = await getMyOrders(session.customerId);
  return NextResponse.json(orders);
}
