import { test, expect } from "@playwright/test";

// 시각 회귀 — 기준 스냅샷과 픽셀 비교. 최초 실행은 --update-snapshots 로 기준 생성.
// 스냅샷은 플랫폼별(폰트 렌더 차이)로 저장되므로, 같은 OS 에서 비교해야 한다.

test("시각 회귀: 로그인 폼(정적)", async ({ page }) => {
  await page.goto("/login");
  await page.waitForLoadState("networkidle");
  // 폼 카드만 — 가장 결정적인 영역.
  await expect(page.locator("main")).toHaveScreenshot("login-main.png");
});

test("시각 회귀: 홈 상품 그리드", async ({ page }) => {
  await page.goto("/");
  await page.waitForLoadState("networkidle");
  await expect(page.locator("main")).toHaveScreenshot("home-main.png", {
    maxDiffPixelRatio: 0.01, // 앤티앨리어싱 미세차 허용
  });
});
