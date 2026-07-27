import { describe, it, expect } from "vitest";
import { productVisual } from "./product-visual";

describe("productVisual", () => {
  it("상품 이름으로 아이콘을 고른다", () => {
    expect(productVisual({ product_id: "x", name: "무선 이어폰" }).glyph).toBe("🎧");
    expect(productVisual({ product_id: "x", name: "기계식 키보드" }).glyph).toBe("⌨️");
    expect(productVisual({ product_id: "x", name: "USB-C 멀티 허브" }).glyph).toBe("🔌");
  });

  it("모르는 상품은 기본 아이콘(📦)", () => {
    expect(productVisual({ product_id: "x", name: "이상한 물건" }).glyph).toBe("📦");
  });

  it("같은 product_id 는 항상 같은 그라디언트(결정적)", () => {
    const a = productVisual({ product_id: "prod-A", name: "이름1" }).gradient;
    const b = productVisual({ product_id: "prod-A", name: "이름2" }).gradient;
    expect(a).toBe(b);
    expect(a).toMatch(/^from-\S+ to-\S+$/);
  });

  it("다른 id 는 (대체로) 다른 그라디언트를 고른다", () => {
    const ids = ["prod-A", "prod-B", "prod-C", "prod-D"];
    const grads = new Set(ids.map((id) => productVisual({ product_id: id, name: "x" }).gradient));
    expect(grads.size).toBeGreaterThan(1);
  });
});
