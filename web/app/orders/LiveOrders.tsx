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

// 아직 움직이는 중인(전이) 상태 — 하나라도 있으면 폴링을 계속한다.
const TRANSIENT = ["PLACED", "CONFIRMED", "RETURN_REQUESTED"];

export default function LiveOrders({ initial }: { initial: OrderView[] }) {
  const [orders, setOrders] = useState(initial);
  const [live, setLive] = useState(false);

  // 사가가 진행 중이면 2초마다 상태를 당겨와 자동 갱신하고, 모두 종결되면 멈춘다.
  useEffect(() => {
    if (!orders.some((o) => TRANSIENT.includes(o.status))) return;
    let active = true;
    setLive(true);
    const tick = async () => {
      const res = await fetch("/api/orders");
      if (!active) return;
      if (res.ok) {
        const data: OrderView[] = await res.json();
        setOrders(data);
        if (!data.some((o) => TRANSIENT.includes(o.status))) {
          setLive(false);
          return;
        }
      }
      if (active) setTimeout(tick, 2000);
    };
    const t = setTimeout(tick, 2000);
    return () => {
      active = false;
      clearTimeout(t);
    };
  }, [orders]);

  async function requestReturn(orderId: string) {
    await fetch(`/api/orders/${orderId}/return`, { method: "POST" });
    const res = await fetch("/api/orders"); // 즉시 한 번 당겨 UI 반영(이후 폴링이 이어감)
    if (res.ok) setOrders(await res.json());
  }

  return (
    <div>
      <div className="mb-3 flex items-center gap-2 text-xs text-neutral-500">
        {live && (
          <span className="flex items-center gap-1.5 text-emerald-400">
            <span className="h-2 w-2 animate-pulse rounded-full bg-emerald-400" />
            실시간 갱신 중
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
