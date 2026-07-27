"use client";

import { useRouter } from "next/navigation";

// 검색어·정렬을 URL 쿼리로 반영한다(SSR 필터 + 공유 가능한 링크).
export default function ProductControls({ q, sort }: { q: string; sort: string }) {
  const router = useRouter();

  function go(nextQ: string, nextSort: string) {
    const params = new URLSearchParams();
    if (nextQ) params.set("q", nextQ);
    if (nextSort) params.set("sort", nextSort);
    const qs = params.toString();
    router.push(qs ? `/?${qs}` : "/");
  }

  return (
    <div className="mb-8 flex gap-3">
      <form
        className="flex-1"
        onSubmit={(e) => {
          e.preventDefault();
          const fd = new FormData(e.currentTarget);
          go(String(fd.get("q") ?? "").trim(), sort);
        }}
      >
        <input
          name="q"
          defaultValue={q}
          aria-label="상품 검색"
          placeholder="상품 검색… (엔터)"
          className="w-full rounded-lg border border-white/10 bg-white/5 px-4 py-2"
        />
      </form>
      <select
        value={sort}
        onChange={(e) => go(q, e.target.value)}
        aria-label="정렬"
        className="rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm"
      >
        <option value="">기본순</option>
        <option value="price-asc">가격 낮은순</option>
        <option value="price-desc">가격 높은순</option>
      </select>
    </div>
  );
}
