import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const { refresh } = vi.hoisted(() => ({ refresh: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh, push: vi.fn() }) }));

import ReturnButton from "./ReturnButton";

describe("<ReturnButton>", () => {
  it("클릭하면 반품 BFF 를 호출하고 화면을 새로고침한다", async () => {
    const f = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", f);

    render(<ReturnButton orderId="order-1" />);
    await userEvent.click(screen.getByRole("button", { name: "반품 요청" }));

    expect(f).toHaveBeenCalledWith("/api/orders/order-1/return", { method: "POST" });
    expect(refresh).toHaveBeenCalled();
  });

  it("실패하면 에러 메시지를 보여준다", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        json: async () => ({ error: "배송된 주문만 반품할 수 있습니다" }),
      }),
    );

    render(<ReturnButton orderId="order-1" />);
    await userEvent.click(screen.getByRole("button", { name: "반품 요청" }));

    expect(await screen.findByText("배송된 주문만 반품할 수 있습니다")).toBeInTheDocument();
  });
});
