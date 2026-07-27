import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { OrderView } from "@/lib/api";

import LiveOrders from "./LiveOrders";

// jsdom 엔 EventSource 가 없으므로 목으로 대체하고, 인스턴스를 붙잡아 이벤트를 흉내낸다.
const { instances } = vi.hoisted(() => ({ instances: [] as MockES[] }));
class MockES {
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;
  constructor(url: string) {
    this.url = url;
    instances.push(this);
  }
  close() {
    this.closed = true;
  }
}

const order = (status: string): OrderView => ({
  order_id: "o1",
  customer_id: "cust-1",
  status,
  total: 39000,
  items: [{ product_id: "prod-A", quantity: 1 }],
});

beforeEach(() => {
  instances.length = 0;
  vi.stubGlobal("EventSource", MockES);
});

describe("<LiveOrders> (SSE)", () => {
  it("마운트하면 SSE 스트림에 연결한다", () => {
    render(<LiveOrders initial={[order("SHIPPED")]} />);
    expect(instances[0]?.url).toBe("/api/orders/stream");
  });

  it("연결되면 '실시간 연결됨'을 표시한다", () => {
    render(<LiveOrders initial={[order("PLACED")]} />);
    act(() => instances[0].onopen?.());
    expect(screen.getByText(/실시간 연결됨/)).toBeInTheDocument();
  });

  it("SSE 메시지로 주문 상태가 갱신된다", () => {
    render(<LiveOrders initial={[order("PLACED")]} />);
    expect(screen.getByText("주문 접수")).toBeInTheDocument();
    act(() => instances[0].onmessage?.({ data: JSON.stringify([order("SHIPPED")]) }));
    expect(screen.getByText("배송 중")).toBeInTheDocument();
  });

  it("SHIPPED 주문의 반품 버튼 → 반품 POST", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) }));
    render(<LiveOrders initial={[order("SHIPPED")]} />);
    await userEvent.click(screen.getByRole("button", { name: "반품 요청" }));
    expect(fetch).toHaveBeenCalledWith("/api/orders/o1/return", { method: "POST" });
  });
});
