import { NextResponse } from "next/server";
import { addProduct } from "@/lib/api";
import { apiError, unauthorized, forbidden } from "@/lib/http";
import { getSession } from "@/lib/session";

// 상품 등록 — 관리자만. product_id 는 서버가 생성한다.
export async function POST(req: Request) {
  const session = await getSession();
  if (!session) return unauthorized();
  if (session.role !== "admin") return forbidden();

  const { name, price } = await req.json();
  try {
    const productId = "prod-" + crypto.randomUUID().slice(0, 8);
    await addProduct(productId, String(name), Number(price));
    return NextResponse.json({ ok: true, product_id: productId });
  } catch (e) {
    return apiError(e);
  }
}
