import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import ProductThumb from "./ProductThumb";

describe("<ProductThumb>", () => {
  it("상품 이름에 맞는 아이콘을 렌더한다", () => {
    render(<ProductThumb product={{ product_id: "prod-A", name: "무선 이어폰" }} />);
    expect(screen.getByText("🎧")).toBeInTheDocument();
  });

  it("그라디언트 배경 클래스를 입힌다", () => {
    const { container } = render(
      <ProductThumb product={{ product_id: "prod-A", name: "무선 이어폰" }} />,
    );
    expect(container.firstChild).toHaveClass("bg-gradient-to-br");
  });
});
