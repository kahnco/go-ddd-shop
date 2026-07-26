import { NextResponse } from "next/server";
import { services } from "@/lib/api";
import { getSession } from "@/lib/session";

export async function POST(req: Request) {
  const session = await getSession();
  if (!session) {
    return NextResponse.json({ error: "로그인이 필요합니다" }, { status: 401 });
  }
  const { product_id } = await req.json();
  const res = await fetch(
    `${services.cart}/carts/${encodeURIComponent(session.customerId)}/items/${encodeURIComponent(product_id)}`,
    {
      method: "DELETE",
      headers: { Authorization: `Bearer ${session.token}` },
    },
  );
  if (!res.ok) {
    const e = await res.json().catch(() => ({}));
    return NextResponse.json({ error: e.error ?? "삭제 실패" }, { status: res.status });
  }
  return NextResponse.json({ ok: true });
}
