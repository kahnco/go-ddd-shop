"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export default function PriceEditor({ productId, price }: { productId: string; price: number }) {
  const router = useRouter();
  const [value, setValue] = useState(String(price));
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);

  async function save() {
    setBusy(true);
    setSaved(false);
    const res = await fetch(`/api/admin/products/${productId}/price`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ price: Number(value) }),
    });
    setBusy(false);
    if (res.ok) {
      setSaved(true);
      router.refresh();
    }
  }

  return (
    <div className="flex items-center gap-2">
      <input
        type="number"
        min={0}
        aria-label={`${productId} 가격`}
        value={value}
        onChange={(e) => {
          setValue(e.target.value);
          setSaved(false);
        }}
        className="w-28 rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-sm"
      />
      <button
        onClick={save}
        disabled={busy || value === String(price)}
        className="rounded-lg border border-white/15 px-3 py-1.5 text-xs text-neutral-300 hover:border-blue-500/40 hover:text-blue-300 disabled:opacity-40"
      >
        저장
      </button>
      {saved && <span className="text-xs text-emerald-400">✓</span>}
    </div>
  );
}
