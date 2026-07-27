"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

export default function LogoutButton() {
  const router = useRouter();
  const [busy, setBusy] = useState(false);

  async function logout() {
    setBusy(true);
    await fetch("/api/auth/logout", { method: "POST" });
    setBusy(false);
    router.push("/");
    router.refresh(); // 서버 컴포넌트(헤더)가 세션을 다시 읽도록
  }

  return (
    <button
      onClick={logout}
      disabled={busy}
      className="text-sm text-neutral-400 transition-colors hover:text-white disabled:opacity-50"
    >
      로그아웃
    </button>
  );
}
