"use client";

import { useEffect, useState } from "react";
import { won, statusLabel } from "@/lib/format";
import type { OrderView } from "@/lib/api";

const statusColor: Record<string, string> = {
  PLACED: "text-neutral-300 bg-white/5",
  CONFIRMED: "text-blue-300 bg-blue-500/10",
  SHIPPED: "text-emerald-300 bg-emerald-500/10",
  CANCELLED: "text-red-300 bg-red-500/10",
  RETURN_REQUESTED: "text-amber-300 bg-amber-500/10",
  REFUNDED: "text-neutral-400 bg-white/5",
};

export default function LiveOrders({ initial }: { initial: OrderView[] }) {
  const [orders, setOrders] = useState(initial);
  const [live, setLive] = useState(false);

  // SSE 구독 — 서버(readmodel)가 주문 이벤트를 밀어 준다. 폴링 없음.
  useEffect(() => {
    const es = new EventSource("/api/orders/stream");
    es.onopen = () => setLive(true);
    es.onmessage = (e) => {
      try {
        setOrders(JSON.parse(e.data) as OrderView[]);
      } catch {
        /* ping/주석 라인 등은 무시 */
      }
    };
    es.onerror = () => setLive(false); // 끊기면 EventSource 가 자동 재연결한다
    return () => es.close();
  }, []);

  async function requestReturn(orderId: string) {
    // 반품만 트리거 — 이후 갱신은 SSE 가 밀어 준다(수동 재조회 불필요).
    await fetch(`/api/orders/${orderId}/return`, { method: "POST" });
  }

  return (
    <div>
      <div className="mb-3 flex items-center gap-2 text-xs text-neutral-500">
        {live && (
          <span className="flex items-center gap-1.5 text-emerald-400">
            <span className="h-2 w-2 animate-pulse rounded-full bg-emerald-400" />
            실시간 연결됨 (SSE)
          </span>
        )}
      </div>
      <div className="space-y-3">
        {orders.map((o) => (
          <div
            key={o.order_id}
            className="flex items-center justify-between rounded-xl border border-white/10 bg-white/[0.03] px-5 py-4"
          >
            <div>
              <p className="font-mono text-sm text-neutral-300">{o.order_id}</p>
              <p className="mt-1 text-xs text-neutral-500">
                상품 {o.items?.length ?? 0}종{o.channel ? ` · ${o.channel}` : ""}
              </p>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-sm text-neutral-300">{won(o.total)}</span>
              <span
                className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${statusColor[o.status] ?? "bg-white/5 text-neutral-300"}`}
              >
                {statusLabel(o.status)}
              </span>
              {o.status === "SHIPPED" && (
                <button
                  onClick={() => requestReturn(o.order_id)}
                  className="rounded-lg border border-white/15 px-3 py-1.5 text-xs text-neutral-300 transition-colors hover:border-amber-500/40 hover:text-amber-300"
                >
                  반품 요청
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
