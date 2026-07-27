import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

// 실제 브라우저 렌더 위에서 axe 로 접근성을 스캔한다.
// color-contrast 는 다크 테마의 의도된 보조 텍스트(muted gray)라 별도 관리 → 제외.
const publicPages = [
  { name: "홈", path: "/" },
  { name: "로그인", path: "/login" },
  { name: "상품 상세", path: "/products/prod-A" },
];

for (const p of publicPages) {
  test(`a11y 위반 없음: ${p.name}`, async ({ page }) => {
    await page.goto(p.path);
    await page.waitForLoadState("networkidle");
    const results = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa"])
      .disableRules(["color-contrast"])
      .analyze();
    // 실패 시 규칙 id·대상이 메시지에 보이도록 요약해서 비교한다.
    const summary = results.violations.map((v) => ({
      id: v.id,
      help: v.help,
      nodes: v.nodes.map((n) => n.target),
    }));
    expect(summary).toEqual([]);
  });
}
