"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export default function AddProductForm() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [price, setPrice] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr("");
    const res = await fetch("/api/admin/products", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, price: Number(price) }),
    });
    setBusy(false);
    if (!res.ok) {
      const e2 = await res.json().catch(() => ({}));
      setErr(e2.error ?? "등록 실패");
      return;
    }
    setName("");
    setPrice("");
    router.refresh();
  }

  return (
    <form onSubmit={submit} className="flex flex-wrap items-end gap-3">
      <input
        aria-label="상품명"
        placeholder="상품명"
        value={name}
        onChange={(e) => setName(e.target.value)}
        required
        className="flex-1 rounded-lg border border-white/10 bg-white/5 px-4 py-2"
      />
      <input
        type="number"
        min={0}
        aria-label="가격"
        placeholder="가격(원)"
        value={price}
        onChange={(e) => setPrice(e.target.value)}
        required
        className="w-32 rounded-lg border border-white/10 bg-white/5 px-4 py-2"
      />
      <button
        type="submit"
        disabled={busy}
        className="rounded-lg bg-blue-600 px-5 py-2 font-medium hover:bg-blue-500 disabled:opacity-50"
      >
        {busy ? "등록 중…" : "상품 등록"}
      </button>
      {err && <span className="w-full text-sm text-red-400">{err}</span>}
    </form>
  );
}
