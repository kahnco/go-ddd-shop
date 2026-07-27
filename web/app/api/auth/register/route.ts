import { NextResponse } from "next/server";
import { register, login } from "@/lib/api";
import { apiError } from "@/lib/http";
import { setSession } from "@/lib/session";

// 가입 후 곧바로 로그인까지 처리해 세션을 세운다(매끄러운 UX).
export async function POST(req: Request) {
  const { email, password, name } = await req.json();
  try {
    await register(email, password, name);
    const token = await login(email, password);
    const s = await setSession(token);
    return NextResponse.json({ ok: true, customerId: s.customerId });
  } catch (e) {
    return apiError(e);
  }
}
