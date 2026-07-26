import { NextResponse } from "next/server";
import { services } from "@/lib/api";
import { setSession } from "@/lib/session";

// 가입 후 곧바로 로그인까지 처리해 세션을 세운다(매끄러운 UX).
export async function POST(req: Request) {
  const { email, password, name } = await req.json();

  const reg = await fetch(`${services.customer}/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password, name }),
  });
  if (!reg.ok) {
    const e = await reg.json().catch(() => ({}));
    return NextResponse.json({ error: e.error ?? "가입 실패" }, { status: reg.status });
  }

  const login = await fetch(`${services.customer}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!login.ok) {
    return NextResponse.json({ error: "가입은 됐지만 자동 로그인 실패" }, { status: 500 });
  }
  const { token } = await login.json();
  const s = await setSession(token);
  return NextResponse.json({ ok: true, customerId: s.customerId });
}
