import { NextResponse } from "next/server";
import { removeCartItem } from "@/lib/api";
import { apiError, unauthorized } from "@/lib/http";
import { getSession } from "@/lib/session";

export async function POST(req: Request) {
  const session = await getSession();
  if (!session) return unauthorized();
  const { product_id } = await req.json();
  try {
    await removeCartItem(session.customerId, session.token, product_id);
    return NextResponse.json({ ok: true });
  } catch (e) {
    return apiError(e);
  }
}
