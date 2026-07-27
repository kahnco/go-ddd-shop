"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export default function AddToCart({ productId }: { productId: string }) {
  const [qty, setQty] = useState(1);
  const [msg, setMsg] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  async function add() {
    setLoading(true);
    setMsg("");
    const res = await fetch("/api/cart/add", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ product_id: productId, quantity: qty }),
    });
    setLoading(false);
    if (res.status === 401) {
      router.push("/login");
      return;
    }
    if (!res.ok) {
      const e = await res.json().catch(() => ({}));
      setMsg(e.error ?? "담기에 실패했습니다");
      return;
    }
    setMsg("장바구니에 담았습니다 ✓");
  }

  return (
    <div className="mt-8 flex items-center gap-3">
      <input
        type="number"
        min={1}
        aria-label="수량"
        value={qty}
        onChange={(e) => setQty(Math.max(1, Number(e.target.value)))}
        className="w-20 rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-center"
      />
      <button
        onClick={add}
        disabled={loading}
        className="rounded-lg bg-blue-600 px-5 py-2 font-medium transition-colors hover:bg-blue-500 disabled:opacity-50"
      >
        {loading ? "담는 중…" : "장바구니 담기"}
      </button>
      {msg && <span className="text-sm text-neutral-400">{msg}</span>}
    </div>
  );
}
