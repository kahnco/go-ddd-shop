import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const { refresh } = vi.hoisted(() => ({ refresh: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh, push: vi.fn() }) }));

import PriceEditor from "./PriceEditor";

describe("<PriceEditor>", () => {
  it("값이 그대로면 저장 버튼이 비활성", () => {
    render(<PriceEditor productId="prod-A" price={1000} />);
    expect(screen.getByRole("button", { name: "저장" })).toBeDisabled();
  });

  it("가격 바꾸고 저장 → PUT 호출", async () => {
    const f = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", f);

    render(<PriceEditor productId="prod-A" price={1000} />);
    const input = screen.getByLabelText("prod-A 가격");
    await userEvent.clear(input);
    await userEvent.type(input, "2000");
    await userEvent.click(screen.getByRole("button", { name: "저장" }));

    expect(f).toHaveBeenCalledWith(
      "/api/admin/products/prod-A/price",
      expect.objectContaining({ method: "PUT" }),
    );
    expect(JSON.parse(f.mock.calls[0][1].body)).toEqual({ price: 2000 });
    await waitFor(() => expect(refresh).toHaveBeenCalled());
  });
});
