import { NextResponse } from "next/server";
import { services } from "@/lib/api";
import { getSession } from "@/lib/session";

// 장바구니 담기. 세션 토큰을 그대로 장바구니 서비스로 전달(신원 전파).
export async function POST(req: Request) {
  const session = await getSession();
  if (!session) {
    return NextResponse.json({ error: "로그인이 필요합니다" }, { status: 401 });
  }
  const { product_id, quantity } = await req.json();
  const res = await fetch(
    `${services.cart}/carts/${encodeURIComponent(session.customerId)}/items`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${session.token}`,
      },
      body: JSON.stringify({ product_id, quantity: Number(quantity) || 1 }),
    },
  );
  if (!res.ok) {
    const e = await res.json().catch(() => ({}));
    return NextResponse.json({ error: e.error ?? "담기 실패" }, { status: res.status });
  }
  return NextResponse.json({ ok: true });
}
