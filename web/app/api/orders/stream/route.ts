import { services } from "@/lib/api";
import { getSession } from "@/lib/session";

// 세션 확인 후 쿠키를 못 다루는 EventSource 대신, 이 BFF 가 대신 인증하고
// readmodel 의 SSE 스트림을 브라우저로 그대로 파이프한다. 브라우저는 web 하고만 통신.
export const dynamic = "force-dynamic";

export async function GET(req: Request) {
  const session = await getSession();
  if (!session) return new Response("unauthorized", { status: 401 });

  const upstream = await fetch(
    `${services.readmodel}/orders/stream?customer=${encodeURIComponent(session.customerId)}`,
    { headers: { Accept: "text/event-stream" }, signal: req.signal }, // 브라우저 끊기면 상류도 중단
  );

  return new Response(upstream.body, {
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache, no-transform",
      Connection: "keep-alive",
    },
  });
}
