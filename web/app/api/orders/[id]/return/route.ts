import { NextResponse } from "next/server";
import { requestReturn } from "@/lib/api";
import { apiError, unauthorized } from "@/lib/http";
import { getSession } from "@/lib/session";

// 반품 요청. 세션 토큰을 주문 서비스로 전달 → 소유 검증 후 SHIPPED → RETURN_REQUESTED,
// 이어서 환불 사가(결제 환불·재고 복원)가 돈다(11편).
export async function POST(_req: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await getSession();
  if (!session) return unauthorized();
  const { id } = await params;
  try {
    await requestReturn(id, session.token);
    return NextResponse.json({ ok: true });
  } catch (e) {
    return apiError(e);
  }
}
