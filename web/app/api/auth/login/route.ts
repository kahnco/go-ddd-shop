import { NextResponse } from "next/server";
import { services } from "@/lib/api";
import { setSession } from "@/lib/session";

// BFF: 브라우저 → (이 Route Handler) → 회원 서비스. 성공 시 토큰을 httpOnly 쿠키로.
export async function POST(req: Request) {
  const { email, password } = await req.json();
  const res = await fetch(`${services.customer}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const e = await res.json().catch(() => ({}));
    return NextResponse.json({ error: e.error ?? "로그인 실패" }, { status: res.status });
  }
  const { token } = await res.json();
  const s = await setSession(token);
  return NextResponse.json({ ok: true, customerId: s.customerId });
}
