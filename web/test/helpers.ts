// 테스트 공용 헬퍼.

// sub·role 을 담은 가짜 JWT — 세션 디코드(setSession)용. 서명은 검증 안 하므로 아무 값.
export function fakeToken(sub: string, role = ""): string {
  const payload = Buffer.from(JSON.stringify({ sub, role, iat: 0, exp: 9_999_999_999 })).toString(
    "base64url",
  );
  return `eyJhbGciOiJIUzI1NiJ9.${payload}.sig`;
}

// fetch 목 응답 하나 만들기(apiFetch 는 res.text() 를 읽는다).
export function fetchResp(body: unknown, ok = true, status = ok ? 200 : 400) {
  return {
    ok,
    status,
    text: async () => (body === undefined ? "" : JSON.stringify(body)),
  } as Response;
}
