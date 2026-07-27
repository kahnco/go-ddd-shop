import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const { refresh } = vi.hoisted(() => ({ refresh: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh, push: vi.fn() }) }));

import CartActions, { type Row } from "./CartActions";

const rows: Row[] = [{ product_id: "prod-A", name: "무선 이어폰", price: 39000, quantity: 2 }];

describe("<CartActions>", () => {
  it("상품명·단가·합계를 렌더한다", () => {
    render(<CartActions rows={rows} total={78000} />);
    expect(screen.getByText("무선 이어폰")).toBeInTheDocument();
    expect(screen.getByText("₩39,000 × 2")).toBeInTheDocument();
    // 소계와 합계 모두 ₩78,000(1행이라 값이 같다) → 두 곳에 렌더되는지 확인.
    expect(screen.getAllByText("₩78,000").length).toBeGreaterThanOrEqual(2);
  });

  it("결제 클릭 → /api/checkout 호출 후 주문번호를 보여준다", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ok: true, order_id: "order-7" }) }),
    );

    render(<CartActions rows={rows} total={78000} />);
    await userEvent.click(screen.getByRole("button", { name: "결제하기" }));

    expect(await screen.findByText(/order-7/)).toBeInTheDocument();
  });

  it("삭제 클릭 → /api/cart/remove 호출 + 새로고침", async () => {
    const f = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", f);

    render(<CartActions rows={rows} total={78000} />);
    await userEvent.click(screen.getByRole("button", { name: "삭제" }));

    expect(f).toHaveBeenCalledWith("/api/cart/remove", expect.objectContaining({ method: "POST" }));
    expect(refresh).toHaveBeenCalled();
  });
});
