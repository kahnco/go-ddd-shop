import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const { refresh } = vi.hoisted(() => ({ refresh: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh, push: vi.fn() }) }));

import AddProductForm from "./AddProductForm";

describe("<AddProductForm>", () => {
  it("등록 → /api/admin/products 로 상품명·가격 전송 후 새로고침", async () => {
    const f = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", f);

    render(<AddProductForm />);
    await userEvent.type(screen.getByLabelText("상품명"), "무선 마우스");
    await userEvent.type(screen.getByLabelText("가격"), "25000");
    await userEvent.click(screen.getByRole("button", { name: "상품 등록" }));

    expect(f).toHaveBeenCalledWith("/api/admin/products", expect.objectContaining({ method: "POST" }));
    expect(JSON.parse(f.mock.calls[0][1].body)).toEqual({ name: "무선 마우스", price: 25000 });
    await waitFor(() => expect(refresh).toHaveBeenCalled());
  });
});
