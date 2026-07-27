import { NextResponse } from "next/server";
import { changePrice } from "@/lib/api";
import { apiError, unauthorized, forbidden } from "@/lib/http";
import { getSession } from "@/lib/session";

// 가격 변경 — 관리자만.
export async function PUT(req: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await getSession();
  if (!session) return unauthorized();
  if (session.role !== "admin") return forbidden();

  const { id } = await params;
  const { price } = await req.json();
  try {
    await changePrice(id, Number(price));
    return NextResponse.json({ ok: true });
  } catch (e) {
    return apiError(e);
  }
}
