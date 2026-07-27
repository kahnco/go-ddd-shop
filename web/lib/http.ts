import { NextResponse } from "next/server";
import { ApiError } from "./api";

// 라우트 핸들러 공통 에러 응답. ApiError 면 그 status 를, 아니면 500 을 돌려준다.
export function apiError(e: unknown): NextResponse {
  const status = e instanceof ApiError ? e.status : 500;
  const message = e instanceof Error ? e.message : "서버 오류";
  return NextResponse.json({ error: message }, { status });
}

// 로그인 안 된 요청용 401.
export function unauthorized(): NextResponse {
  return NextResponse.json({ error: "로그인이 필요합니다" }, { status: 401 });
}
