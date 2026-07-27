import { NextResponse } from "next/server";
import { addCartItem } from "@/lib/api";
import { apiError, unauthorized } from "@/lib/http";
import { getSession } from "@/lib/session";

// 장바구니 담기. 세션 토큰을 그대로 장바구니 서비스로 전달(신원 전파).
export async function POST(req: Request) {
  const session = await getSession();
  if (!session) return unauthorized();
  const { product_id, quantity } = await req.json();
  try {
    await addCartItem(session.customerId, session.token, product_id, Number(quantity) || 1);
    return NextResponse.json({ ok: true });
  } catch (e) {
    return apiError(e);
  }
}
