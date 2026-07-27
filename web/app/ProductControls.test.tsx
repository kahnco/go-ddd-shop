import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const { push } = vi.hoisted(() => ({ push: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));

import ProductControls from "./ProductControls";

describe("<ProductControls>", () => {
  it("검색어 엔터 → URL 쿼리로 이동", async () => {
    render(<ProductControls q="" sort="" />);
    await userEvent.type(screen.getByLabelText("상품 검색"), "이어폰{Enter}");
    expect(push).toHaveBeenCalledWith("/?q=%EC%9D%B4%EC%96%B4%ED%8F%B0");
  });

  it("정렬 변경 → URL 쿼리로 이동(기존 검색어 유지)", async () => {
    render(<ProductControls q="key" sort="" />);
    await userEvent.selectOptions(screen.getByLabelText("정렬"), "price-asc");
    expect(push).toHaveBeenCalledWith("/?q=key&sort=price-asc");
  });
});
