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

function login(customerId: string) {
  h.store.map.set(
    "shop_session",
    JSON.stringify({ token: fakeToken(customerId), customerId }),
  );
}

beforeEach(() => h.store.map.clear());

describe("POST /api/checkout", () => {
  it("로그인 안 되어 있으면 401 (백엔드 호출도 안 함)", async () => {
    const f = vi.fn();
    vi.stubGlobal("fetch", f);

    const res = await POST();

    expect(res.status).toBe(401);
    expect(f).not.toHaveBeenCalled();
  });

  it("로그인 상태면 결제 → order_id 반환", async () => {
    login("cust-1");
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(fetchResp({ order_id: "order-9" })));

    const res = await POST();

    expect(res.status).toBe(200);
    await expect(res.json()).resolves.toMatchObject({ ok: true, order_id: "order-9" });
  });

  it("장바구니가 비면 백엔드 오류 status 를 그대로 전달", async () => {
    login("cust-1");
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(fetchResp({ error: "빈 장바구니" }, false, 400)));

    const res = await POST();

    expect(res.status).toBe(400);
  });
});
