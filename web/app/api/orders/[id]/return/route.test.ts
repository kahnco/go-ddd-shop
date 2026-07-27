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

const ctx = { params: Promise.resolve({ id: "order-1" }) };
const req = () => new Request("http://x/api/orders/order-1/return", { method: "POST" });

beforeEach(() => h.store.map.clear());

describe("POST /api/orders/[id]/return", () => {
  it("비로그인 → 401", async () => {
    const res = await POST(req(), ctx);
    expect(res.status).toBe(401);
  });

  it("배송 안 된 주문 반품 시 백엔드 409 를 그대로 전달", async () => {
    h.store.map.set(
      "shop_session",
      JSON.stringify({ token: fakeToken("cust-1"), customerId: "cust-1" }),
    );
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(fetchResp({ error: "허용되지 않은 상태 전이" }, false, 409)),
    );

    const res = await POST(req(), ctx);

    expect(res.status).toBe(409);
    await expect(res.json()).resolves.toMatchObject({ error: "허용되지 않은 상태 전이" });
  });
});
