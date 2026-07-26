import Link from "next/link";
import { listProducts, won } from "@/lib/api";

// 서버 컴포넌트 — 카탈로그를 서버(컨테이너)에서 직접 호출해 SSR 로 그린다.
// 브라우저는 완성된 HTML 을 받으므로 CORS 도, 로딩 스피너도 필요 없다.
export default async function Home() {
  const products = await listProducts();

  return (
    <div>
      <h1 className="mb-2 text-2xl font-bold">상품</h1>
      <p className="mb-8 text-sm text-neutral-500">
        카탈로그 서비스에서 서버 렌더링으로 불러온 상품 목록입니다.
      </p>

      {products.length === 0 ? (
        <div className="rounded-xl border border-white/10 bg-white/[0.03] p-8 text-center text-neutral-500">
          아직 상품이 없습니다. 카탈로그에 상품을 등록해 주세요.
          <br />
          <code className="mt-2 inline-block text-xs text-neutral-600">
            POST /products {"{ product_id, name, price }"}
          </code>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {products.map((p) => (
            <Link
              key={p.product_id}
              href={`/products/${p.product_id}`}
              className="group rounded-xl border border-white/10 bg-white/[0.03] p-5 transition-colors hover:border-blue-500/30 hover:bg-blue-500/[0.06]"
            >
              <div className="mb-6 flex aspect-square items-center justify-center rounded-lg bg-white/5 text-4xl">
                📦
              </div>
              <h2 className="font-semibold">{p.name}</h2>
              <p className="mt-1 text-sm text-neutral-400">{won(p.price)}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
