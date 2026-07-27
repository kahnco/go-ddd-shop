import { describe, it, expect, vi } from "vitest";
import { login, checkout, addCartItem, getCart, requestReturn } from "./api";
import { fetchResp } from "@/test/helpers";

describe("api 어댑터", () => {
  it("login: 성공하면 토큰을 반환한다", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(fetchResp({ token: "tok-1" })));
    await expect(login("a@b.com", "pw")).resolves.toBe("tok-1");
  });

  it("login: 실패하면 status 를 담은 ApiError 를 던진다", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(fetchResp({ error: "틀림" }, false, 401)));
    await expect(login("a@b.com", "pw")).rejects.toMatchObject({ status: 401, message: "틀림" });
  });

  it("checkout: order_id 를 반환하고 Bearer 를 붙인다", async () => {
    const f = vi.fn().mockResolvedValue(fetchResp({ order_id: "order-9" }));
    vi.stubGlobal("fetch", f);
    await expect(checkout("cust-1", "tok")).resolves.toBe("order-9");
    const init = f.mock.calls[0][1] as RequestInit;
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer tok");
  });

  it("addCartItem: 올바른 URL·바디로 POST 한다", async () => {
    const f = vi.fn().mockResolvedValue(fetchResp({}));
    vi.stubGlobal("fetch", f);
    await addCartItem("cust-1", "tok", "prod-A", 2);
    const [url, init] = f.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/carts/cust-1/items");
    expect(JSON.parse(init.body as string)).toEqual({ product_id: "prod-A", quantity: 2 });
  });

  it("requestReturn: 반품 엔드포인트로 POST", async () => {
    const f = vi.fn().mockResolvedValue(fetchResp({}));
    vi.stubGlobal("fetch", f);
    await requestReturn("order-1", "tok");
    expect(f.mock.calls[0][0]).toContain("/orders/order-1/return");
  });

  it("getCart: 백엔드가 실패해도 빈 장바구니로 폴백한다(SSR 안전)", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(fetchResp({}, false, 500)));
    await expect(getCart("cust-1", "tok")).resolves.toEqual({
      customer_id: "cust-1",
      items: [],
    });
  });
});
