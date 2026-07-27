import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { OrderView } from "@/lib/api";

import LiveOrders from "./LiveOrders";

const order = (status: string): OrderView => ({
  order_id: "o1",
  customer_id: "cust-1",
  status,
  total: 39000,
  items: [{ product_id: "prod-A", quantity: 1 }],
});

describe("<LiveOrders>", () => {
  it("전이 상태(PLACED)면 '실시간 갱신 중'을 표시한다", () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));
    render(<LiveOrders initial={[order("PLACED")]} />);
    expect(screen.getByText("실시간 갱신 중")).toBeInTheDocument();
    expect(screen.getByText("주문 접수")).toBeInTheDocument();
  });

  it("종결 상태(SHIPPED)면 폴링하지 않고 반품 버튼을 보여준다", async () => {
    const f = vi.fn().mockResolvedValue({ ok: true, json: async () => [order("REFUNDED")] });
    vi.stubGlobal("fetch", f);

    render(<LiveOrders initial={[order("SHIPPED")]} />);
    expect(screen.queryByText("실시간 갱신 중")).not.toBeInTheDocument(); // 폴링 안 함
    expect(screen.getByText("배송 중")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "반품 요청" }));
    expect(f).toHaveBeenCalledWith("/api/orders/o1/return", { method: "POST" });
  });
});
