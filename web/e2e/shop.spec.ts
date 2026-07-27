import { test, expect } from "@playwright/test";

// 실제 브라우저로 쇼핑 전 여정을 돈다(풀스택이 떠 있어야 함).
test("가입 → 상품 탐색 → 담기 → 결제 → 내 주문 → 반품", async ({ page }) => {
  const email = `e2e-${Date.now()}@test.com`;

  // 1) 회원가입(자동 로그인) — 로그인 페이지에서 회원가입 모드로.
  await page.goto("/login");
  await page.getByRole("button", { name: "회원가입" }).click();
  await page.getByPlaceholder("이름").fill("E2E 사용자");
  await page.getByPlaceholder("이메일").fill(email);
  await page.getByPlaceholder(/비밀번호/).fill("supersecret");
  await page.getByRole("button", { name: "회원가입" }).nth(1).click(); // 토글 아닌 제출 버튼
  await expect(page).toHaveURL("http://localhost:3000/");

  // 2) 상품 탐색 → 상세
  await expect(page.getByRole("heading", { name: "상품" })).toBeVisible();
  await page.getByRole("link", { name: /무선 이어폰/ }).click();
  await expect(page.getByRole("heading", { name: "무선 이어폰" })).toBeVisible();

  // 3) 장바구니 담기
  await page.getByRole("button", { name: "장바구니 담기" }).click();
  await expect(page.getByText("장바구니에 담았습니다 ✓")).toBeVisible();

  // 4) 장바구니 → 결제
  await page.goto("/cart");
  await page.waitForLoadState("networkidle"); // 하이드레이션까지 안정화
  await expect(page.getByText("무선 이어폰")).toBeVisible();
  await page.waitForTimeout(800); // 하이드레이션 여유(onClick 부착 대기)
  await page.getByRole("button", { name: "결제하기" }).click();
  // 결제 체인(cart→customer→ordering)이 콜드일 때를 감안해 넉넉히.
  await expect(page.getByText("주문이 접수되었습니다 🎉")).toBeVisible({ timeout: 20_000 });

  // 5) 내 주문
  await page.getByRole("button", { name: "내 주문 보기 →" }).click();
  await expect(page).toHaveURL(/\/orders$/);

  // 6) 사가가 배송까지 진행되길 기다린다(SSR 이라 새로고침으로 갱신).
  await expect(async () => {
    await page.reload();
    await expect(page.getByRole("button", { name: "반품 요청" })).toBeVisible({ timeout: 1500 });
  }).toPass({ timeout: 60_000 });

  // 7) 반품 요청 → 환불 흐름 진입
  await page.getByRole("button", { name: "반품 요청" }).click();
  await expect(async () => {
    await page.reload();
    await expect(page.getByText(/환불 완료|반품 요청/)).toBeVisible({ timeout: 1500 });
  }).toPass({ timeout: 30_000 });
});
