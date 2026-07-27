import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchResp } from "@/test/helpers";

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

function loginAs(role: string) {
  h.store.map.set("shop_session", JSON.stringify({ token: "t", customerId: "u", role }));
}
const req = () =>
  new Request("http://x/api/admin/products", {
    method: "POST",
    body: JSON.stringify({ name: "새 상품", price: 1000 }),
  });

beforeEach(() => h.store.map.clear());

describe("POST /api/admin/products (관리자 게이트)", () => {
  it("비로그인 → 401", async () => {
    const res = await POST(req());
    expect(res.status).toBe(401);
  });

  it("일반 회원 → 403 (백엔드 호출 안 함)", async () => {
    loginAs("customer");
    const f = vi.fn();
    vi.stubGlobal("fetch", f);
    const res = await POST(req());
    expect(res.status).toBe(403);
    expect(f).not.toHaveBeenCalled();
  });

  it("관리자 → 200 + product_id 생성", async () => {
    loginAs("admin");
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(fetchResp({})));
    const res = await POST(req());
    expect(res.status).toBe(200);
    const body = await res.json();
    expect(body.ok).toBe(true);
    expect(body.product_id).toMatch(/^prod-/);
  });
});
