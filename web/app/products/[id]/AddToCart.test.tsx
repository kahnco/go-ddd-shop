import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const { push } = vi.hoisted(() => ({ push: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ push, refresh: vi.fn() }) }));

import AddToCart from "./AddToCart";

describe("<AddToCart>", () => {
  it("담기 클릭 → /api/cart/add 로 상품·수량 전송 + 성공 메시지", async () => {
    const f = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", f);

    render(<AddToCart productId="prod-A" />);
    await userEvent.click(screen.getByRole("button", { name: /장바구니 담기/ }));

    expect(f).toHaveBeenCalledWith("/api/cart/add", expect.objectContaining({ method: "POST" }));
    expect(JSON.parse(f.mock.calls[0][1].body)).toMatchObject({ product_id: "prod-A", quantity: 1 });
    expect(await screen.findByText("장바구니에 담았습니다 ✓")).toBeInTheDocument();
  });

  it("401 이면 로그인 페이지로 보낸다", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 401, json: async () => ({}) }),
    );

    render(<AddToCart productId="prod-A" />);
    await userEvent.click(screen.getByRole("button", { name: /장바구니 담기/ }));

    await waitFor(() => expect(push).toHaveBeenCalledWith("/login"));
  });
});
