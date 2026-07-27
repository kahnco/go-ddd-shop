import Link from "next/link";
import { listProducts, won } from "@/lib/api";
import ProductThumb from "@/app/components/ProductThumb";
import ProductControls from "@/app/ProductControls";

// 서버 컴포넌트 — 카탈로그를 서버(컨테이너)에서 직접 호출해 SSR 로 그린다.
// 검색어(q)·정렬(sort)은 URL 쿼리로 받아 서버에서 필터·정렬한다(공유 가능한 링크).
export default async function Home({
  searchParams,
}: {
  searchParams: Promise<{ q?: string; sort?: string }>;
}) {
  const { q = "", sort = "" } = await searchParams;

  let products = await listProducts(); // 기본은 product_id 정렬
  if (q) {
    const needle = q.toLowerCase();
    products = products.filter((p) => p.name.toLowerCase().includes(needle));
  }
  if (sort === "price-asc") products = [...products].sort((a, b) => a.price - b.price);
  else if (sort === "price-desc") products = [...products].sort((a, b) => b.price - a.price);

  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold">상품</h1>
      <p className="mb-8 text-sm text-neutral-500">
        카탈로그 서비스에서 서버 렌더링으로 불러온 상품 목록입니다.
      </p>

      <ProductControls q={q} sort={sort} />

      {products.length === 0 ? (
        <div className="rounded-xl border border-white/10 bg-white/[0.03] p-8 text-center text-neutral-500">
          {q ? `"${q}" 에 대한 상품이 없습니다.` : "아직 상품이 없습니다."}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {products.map((p) => (
            <Link
              key={p.product_id}
              href={`/products/${p.product_id}`}
              className="group rounded-xl border border-white/10 bg-white/[0.03] p-5 transition-colors hover:border-blue-500/30 hover:bg-blue-500/[0.06]"
            >
              <ProductThumb product={p} className="mb-5 aspect-square" size="text-6xl" />
              <h2 className="font-semibold">{p.name}</h2>
              <p className="mt-1 text-sm text-neutral-400">{won(p.price)}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
