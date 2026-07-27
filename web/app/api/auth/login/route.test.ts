import { describe, it, expect, vi, beforeEach } from "vitest";
import { fakeToken, fetchResp } from "@/test/helpers";

const h = vi.hoisted(() => {
  const map = new Map<string, string>();
  return {
    store: {
      get: (n: string) => (map.has(n) ? { name: n, value: map.get(n)! } : undefined),
      set: (n: string, v: string) => void map.set(n, v),
      delete: (n: string) => void map.delete(n),
      map,
    },
  };
});
vi.mock("next/headers", () => ({ cookies: async () => h.store }));

import { POST } from "./route";

const req = (body: unknown) =>
  new Request("http://x/api/auth/login", { method: "POST", body: JSON.stringify(body) });

beforeEach(() => h.store.map.clear());

describe("POST /api/auth/login", () => {
  it("성공: 200 + 세션 쿠키에 회원 ID 저장", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(fetchResp({ token: fakeToken("cust-1") })));

    const res = await POST(req({ email: "a@b.com", password: "supersecret" }));

    expect(res.status).toBe(200);
    await expect(res.json()).resolves.toMatchObject({ ok: true, customerId: "cust-1" });
    expect(h.store.map.get("shop_session")).toContain("cust-1");
  });

  it("실패: 백엔드 401 을 그대로 401 로 되돌린다(세션 없음)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(fetchResp({ error: "이메일 또는 비밀번호가 올바르지 않습니다" }, false, 401)),
    );

    const res = await POST(req({ email: "a@b.com", password: "wrong" }));

    expect(res.status).toBe(401);
    expect(h.store.map.has("shop_session")).toBe(false);
  });
});
