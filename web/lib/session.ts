import { cookies } from "next/headers";

// 로그인 세션을 httpOnly 쿠키에 담는다. 브라우저 JS 는 토큰을 못 읽고,
// 서버(BFF·서버 컴포넌트)만 쿠키를 꺼내 Go 서비스를 Bearer 로 호출한다.
const COOKIE = "shop_session";

export type Session = { token: string; customerId: string; role: string };

// JWT 페이로드(가운데 조각)에서 sub·role 을 꺼낸다. 검증은 백엔드가 하므로 여기선 디코드만.
function decodeClaims(token: string): { sub: string; role: string } {
  try {
    const payload = token.split(".")[1] ?? "";
    const json = Buffer.from(payload, "base64url").toString("utf8");
    const c = JSON.parse(json);
    return { sub: (c.sub as string) ?? "", role: (c.role as string) ?? "customer" };
  } catch {
    return { sub: "", role: "customer" };
  }
}

export async function getSession(): Promise<Session | null> {
  const raw = (await cookies()).get(COOKIE)?.value;
  if (!raw) return null;
  try {
    const s = JSON.parse(raw) as Session;
    return s.token ? s : null;
  } catch {
    return null;
  }
}

export async function setSession(token: string): Promise<Session> {
  const { sub, role } = decodeClaims(token);
  const s: Session = { token, customerId: sub, role };
  (await cookies()).set(COOKIE, JSON.stringify(s), {
    httpOnly: true,
    sameSite: "lax",
    path: "/",
    maxAge: 60 * 60 * 24, // 24시간(토큰 만료와 맞춤)
  });
  return s;
}

export async function clearSession(): Promise<void> {
  (await cookies()).delete(COOKIE);
}
