import Link from "next/link";
import { notFound } from "next/navigation";
import { getProduct, won } from "@/lib/api";
import ProductThumb from "@/app/components/ProductThumb";
import AddToCart from "./AddToCart";

// Next 15 — 동적 라우트의 params 는 비동기.
export default async function ProductPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const product = await getProduct(id);
  if (!product) notFound();

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
          <p className="mt-6 text-sm leading-relaxed text-neutral-400">
            상품 ID <code className="text-neutral-500">{product.product_id}</code> · 가격은
            카탈로그 서비스가 정하며, 주문 시 서버가 이 가격을 적용합니다(클라이언트 가격 조작
            불가).
          </p>
          <AddToCart productId={product.product_id} />
        </div>
      </div>
    </div>
  );
}
