import { NextResponse } from "next/server";
import { services } from "@/lib/api";
import { getSession } from "@/lib/session";

// 반품 요청. 세션 토큰을 주문 서비스로 전달 → 소유 검증 후 SHIPPED → RETURN_REQUESTED,
// 이어서 환불 사가(결제 환불·재고 복원)가 돈다(11편).
export async function POST(_req: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await getSession();
  if (!session) {
    return NextResponse.json({ error: "로그인이 필요합니다" }, { status: 401 });
  }
  const { id } = await params;
  const res = await fetch(`${services.ordering}/orders/${encodeURIComponent(id)}/return`, {
    method: "POST",
    headers: { Authorization: `Bearer ${session.token}` },
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    return NextResponse.json({ error: body.error ?? "반품 요청 실패" }, { status: res.status });
  }
  return NextResponse.json({ ok: true });
}
