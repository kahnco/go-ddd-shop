"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export default function LoginForm() {
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setErr("");
    const url = mode === "login" ? "/api/auth/login" : "/api/auth/register";
    const body = mode === "login" ? { email, password } : { email, password, name };
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    setLoading(false);
    if (!res.ok) {
      const e2 = await res.json().catch(() => ({}));
      setErr(e2.error ?? "실패했습니다");
      return;
    }
    router.push("/"); // 로그인/가입 성공 → 홈으로
    router.refresh(); // 서버 컴포넌트가 세션을 다시 읽도록
  }

  return (
    <div className="mx-auto max-w-sm">
      <div className="mb-6 flex gap-2 text-sm">
        <button
          onClick={() => setMode("login")}
          className={`rounded-full px-4 py-1.5 ${mode === "login" ? "bg-blue-600" : "bg-white/5 text-neutral-400"}`}
        >
          로그인
        </button>
        <button
          onClick={() => setMode("register")}
          className={`rounded-full px-4 py-1.5 ${mode === "register" ? "bg-blue-600" : "bg-white/5 text-neutral-400"}`}
        >
          회원가입
        </button>
      </div>

      <form onSubmit={submit} className="space-y-3">
        {mode === "register" && (
          <input
            aria-label="이름"
            placeholder="이름"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full rounded-lg border border-white/10 bg-white/5 px-4 py-2.5"
          />
        )}
        <input
          type="email"
          aria-label="이메일"
          placeholder="이메일"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          className="w-full rounded-lg border border-white/10 bg-white/5 px-4 py-2.5"
        />
        <input
          type="password"
          aria-label="비밀번호"
          placeholder="비밀번호(8자 이상)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          className="w-full rounded-lg border border-white/10 bg-white/5 px-4 py-2.5"
        />
        {err && <p className="text-sm text-red-400">{err}</p>}
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-lg bg-blue-600 py-2.5 font-medium transition-colors hover:bg-blue-500 disabled:opacity-50"
        >
          {loading ? "처리 중…" : mode === "login" ? "로그인" : "회원가입"}
        </button>
      </form>
    </div>
  );
}
