import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { axe } from "vitest-axe";
import * as matchers from "vitest-axe/matchers";

expect.extend(matchers);

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn(), refresh: vi.fn() }) }));

import LoginForm from "./login/LoginForm";
import AddToCart from "./products/[id]/AddToCart";
import CartActions from "./cart/CartActions";
import ProductThumb from "./components/ProductThumb";

// jsdom 은 레이아웃/색을 계산하지 못하므로 color-contrast 는 끄고(브라우저 axe 에서 별도로),
// 라벨·역할·구조 위반을 잡는다.
const scan = (c: HTMLElement) => axe(c, { rules: { "color-contrast": { enabled: false } } });

describe("접근성(a11y) — 컴포넌트", () => {
  it("LoginForm: 위반 없음", async () => {
    const { container } = render(<LoginForm />);
    expect(await scan(container)).toHaveNoViolations();
  });

  it("AddToCart: 위반 없음", async () => {
    const { container } = render(<AddToCart productId="prod-A" />);
    expect(await scan(container)).toHaveNoViolations();
  });

  it("CartActions: 위반 없음", async () => {
    const { container } = render(
      <CartActions
        rows={[{ product_id: "prod-A", name: "무선 이어폰", price: 39000, quantity: 1 }]}
        total={39000}
      />,
    );
    expect(await scan(container)).toHaveNoViolations();
  });

  it("ProductThumb: 위반 없음", async () => {
    const { container } = render(
      <ProductThumb product={{ product_id: "prod-A", name: "무선 이어폰" }} />,
    );
    expect(await scan(container)).toHaveNoViolations();
  });
});
