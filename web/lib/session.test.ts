import { describe, it, expect, vi, beforeEach } from "vitest";
import { fakeToken } from "@/test/helpers";

// next/headers 의 cookies() 를 인메모리 저장소로 대체.
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

import { getSession, setSession, clearSession } from "./session";

beforeEach(() => h.store.map.clear());

describe("session", () => {
  it("setSession 은 토큰의 sub 를 회원 ID 로 저장한다", async () => {
    const s = await setSession(fakeToken("cust-42"));
    expect(s.customerId).toBe("cust-42");
    expect(h.store.map.get("shop_session")).toContain("cust-42");
  });

  it("getSession 은 저장된 세션을 읽는다", async () => {
    await setSession(fakeToken("cust-7"));
    await expect(getSession()).resolves.toMatchObject({ customerId: "cust-7" });
  });

  it("세션이 없으면 null", async () => {
    await expect(getSession()).resolves.toBeNull();
  });

  it("clearSession 후에는 세션이 사라진다", async () => {
    await setSession(fakeToken("cust-1"));
    await clearSession();
    await expect(getSession()).resolves.toBeNull();
  });
});
