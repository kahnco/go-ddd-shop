import { describe, it, expect } from "vitest";
import { won, statusLabel } from "./format";

describe("won", () => {
  it("천 단위 콤마와 ₩ 를 붙인다", () => {
    expect(won(89000)).toBe("₩89,000");
    expect(won(1000000)).toBe("₩1,000,000");
    expect(won(0)).toBe("₩0");
  });
});

describe("statusLabel", () => {
  it("주문 상태를 한글로 매핑한다", () => {
    expect(statusLabel("PLACED")).toBe("주문 접수");
    expect(statusLabel("SHIPPED")).toBe("배송 중");
    expect(statusLabel("REFUNDED")).toBe("환불 완료");
  });

  it("모르는 상태는 원문 그대로 둔다", () => {
    expect(statusLabel("WHATEVER")).toBe("WHATEVER");
  });
});
