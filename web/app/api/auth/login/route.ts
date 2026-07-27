import { NextResponse } from "next/server";
import { login } from "@/lib/api";
import { apiError } from "@/lib/http";
import { setSession } from "@/lib/session";

// BFF: 브라우저 → (이 핸들러) → 회원 서비스. 성공 시 토큰을 httpOnly 쿠키로.
export async function POST(req: Request) {
  const { email, password } = await req.json();
  try {
    const token = await login(email, password);
    const s = await setSession(token);
    return NextResponse.json({ ok: true, customerId: s.customerId });
  } catch (e) {
    return apiError(e);
  }
}
