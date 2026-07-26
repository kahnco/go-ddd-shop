"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

// 배송 중(SHIPPED) 주문에만 뜨는 반품 요청 버튼.
export default function ReturnButton({ orderId }: { orderId: string }) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  async function request() {
    setBusy(true);
    setErr("");
    const res = await fetch(`/api/orders/${orderId}/return`, { method: "POST" });
    setBusy(false);
    if (!res.ok) {
      const e = await res.json().catch(() => ({}));
      setErr(e.error ?? "실패");
      return;
    }
    router.refresh(); // 상태가 RETURN_REQUESTED → REFUNDED 로 갱신되도록
  }

  return (
    <div className="flex flex-col items-end">
      <button
        onClick={request}
        disabled={busy}
        className="rounded-lg border border-white/15 px-3 py-1.5 text-xs text-neutral-300 transition-colors hover:border-amber-500/40 hover:text-amber-300 disabled:opacity-50"
      >
        {busy ? "요청 중…" : "반품 요청"}
      </button>
      {err && <span className="mt-1 text-xs text-red-400">{err}</span>}
    </div>
  );
}
