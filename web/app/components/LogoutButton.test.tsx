import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const { push, refresh } = vi.hoisted(() => ({ push: vi.fn(), refresh: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ push, refresh }) }));

import LogoutButton from "./LogoutButton";

describe("<LogoutButton>", () => {
  it("클릭 → 로그아웃 BFF 호출 후 홈으로 이동·새로고침", async () => {
    const f = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", f);

    render(<LogoutButton />);
    await userEvent.click(screen.getByRole("button", { name: "로그아웃" }));

    expect(f).toHaveBeenCalledWith("/api/auth/logout", { method: "POST" });
    await waitFor(() => {
      expect(push).toHaveBeenCalledWith("/");
      expect(refresh).toHaveBeenCalled();
    });
  });
});
