"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { won } from "@/lib/format";

export type Row = { product_id: string; name: string; price: number; quantity: number };

export default function CartActions({ rows, total }: { rows: Row[]; total: number }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [orderId, setOrderId] = useState("");
  const [err, setErr] = useState("");

  async function remove(productId: string) {
    setBusy(true);
    await fetch("/api/cart/remove", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ product_id: productId }),
    });
    setBusy(false);
    router.refresh();
  }

  async function checkout() {
    setBusy(true);
    setErr("");
    const res = await fetch("/api/checkout", { method: "POST" });
    setBusy(false);
    const body = await res.json().catch(() => ({}));
    if (!res.ok) {
      setErr(body.error ?? "결제에 실패했습니다");
      return;
    }
    setOrderId(body.order_id);
    router.refresh();
  }

  if (orderId) {
    return (
      <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/[0.06] p-8 text-center">
        <p className="text-lg font-semibold text-emerald-300">주문이 접수되었습니다 🎉</p>
        <p className="mt-1 text-sm text-neutral-400">
          주문번호 <code className="text-neutral-300">{orderId}</code>
        </p>
        <button
          onClick={() => router.push("/orders")}
          className="mt-6 rounded-lg bg-blue-600 px-5 py-2 font-medium hover:bg-blue-500"
        >
          내 주문 보기 →
        </button>
      </div>
    );
  }

  return (
    <div>
      <div className="divide-y divide-white/5 rounded-xl border border-white/10 bg-white/[0.03]">
        {rows.map((r) => (
          <div key={r.product_id} className="flex items-center justify-between px-5 py-4">
            <div>
              <p className="font-medium">{r.name}</p>
              <p className="text-sm text-neutral-500">
                {won(r.price)} × {r.quantity}
              </p>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-sm text-neutral-300">{won(r.price * r.quantity)}</span>
              <button
                onClick={() => remove(r.product_id)}
                disabled={busy}
                className="text-xs text-neutral-500 hover:text-red-400 disabled:opacity-50"
              >
                삭제
              </button>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-6 flex items-center justify-between">
        <span className="text-neutral-400">합계</span>
        <span className="text-xl font-bold">{won(total)}</span>
      </div>

      {err && <p className="mt-3 text-sm text-red-400">{err}</p>}

      <button
        onClick={checkout}
        disabled={busy || rows.length === 0}
        className="mt-6 w-full rounded-lg bg-blue-600 py-3 font-semibold transition-colors hover:bg-blue-500 disabled:opacity-50"
      >
        {busy ? "처리 중…" : "결제하기"}
      </button>
    </div>
  );
}
