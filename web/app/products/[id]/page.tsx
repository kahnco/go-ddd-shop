import Link from "next/link";
import { notFound } from "next/navigation";
import { getProduct, getStock, won } from "@/lib/api";
import ProductThumb from "@/app/components/ProductThumb";
import AddToCart from "./AddToCart";

// Next 15 — 동적 라우트의 params 는 비동기.
export default async function ProductPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const [product, stock] = await Promise.all([getProduct(id), getStock(id)]);
  if (!product) notFound();

  const soldOut = stock <= 0;

  return (
    <div>
      <Link href="/" className="text-sm text-neutral-500 hover:text-white">
        ← 상품 목록
      </Link>

      <div className="mt-6 grid grid-cols-1 gap-8 sm:grid-cols-2">
        <ProductThumb product={product} className="aspect-square" size="text-8xl" />
        <div>
          <h1 className="text-2xl font-bold">{product.name}</h1>
          <p className="mt-2 text-xl text-blue-400">{won(product.price)}</p>

          <p className="mt-3 text-sm">
            {soldOut ? (
              <span className="rounded-full bg-red-500/10 px-3 py-1 font-medium text-red-300">
                품절
              </span>
            ) : (
              <span className="text-neutral-400">
                재고 <span className="text-neutral-200">{stock}</span>개
                {stock <= 3 && <span className="ml-1 text-amber-300">· 곧 소진</span>}
              </span>
            )}
          </p>

          <p className="mt-6 text-sm leading-relaxed text-neutral-400">
            상품 ID <code className="text-neutral-500">{product.product_id}</code> · 가격은
            카탈로그가, 재고는 재고 서비스가 소유합니다.
          </p>

          <AddToCart productId={product.product_id} soldOut={soldOut} maxQty={stock} />
        </div>
      </div>
    </div>
  );
}
