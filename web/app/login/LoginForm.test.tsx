import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const { push } = vi.hoisted(() => ({ push: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ push, refresh: vi.fn() }) }));

import LoginForm from "./LoginForm";

describe("<LoginForm>", () => {
  it("로그인 제출 → /api/auth/login 호출 후 홈으로 이동", async () => {
    const f = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", f);

    render(<LoginForm />);
    await userEvent.type(screen.getByPlaceholderText("이메일"), "a@b.com");
    await userEvent.type(screen.getByPlaceholderText(/비밀번호/), "supersecret{Enter}");

    await waitFor(() => expect(f).toHaveBeenCalledWith("/api/auth/login", expect.anything()));
    expect(JSON.parse(f.mock.calls[0][1].body)).toMatchObject({ email: "a@b.com" });
    expect(push).toHaveBeenCalledWith("/");
  });

  it("회원가입 모드로 전환하면 /api/auth/register 를 호출한다", async () => {
    const f = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", f);

    render(<LoginForm />);
    await userEvent.click(screen.getByRole("button", { name: "회원가입" }));
    await userEvent.type(screen.getByPlaceholderText("이름"), "홍길동");
    await userEvent.type(screen.getByPlaceholderText("이메일"), "a@b.com");
    await userEvent.type(screen.getByPlaceholderText(/비밀번호/), "supersecret{Enter}");

    await waitFor(() => expect(f).toHaveBeenCalledWith("/api/auth/register", expect.anything()));
    expect(JSON.parse(f.mock.calls[0][1].body)).toMatchObject({ name: "홍길동", email: "a@b.com" });
  });

  it("실패하면 에러 메시지를 보여준다", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, json: async () => ({ error: "로그인 실패" }) }),
    );

    render(<LoginForm />);
    await userEvent.type(screen.getByPlaceholderText("이메일"), "a@b.com");
    await userEvent.type(screen.getByPlaceholderText(/비밀번호/), "wrongpass{Enter}");

    expect(await screen.findByText("로그인 실패")).toBeInTheDocument();
  });
});
