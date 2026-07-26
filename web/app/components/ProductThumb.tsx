import { productVisual } from "@/lib/product-visual";

// 상품 썸네일. 실제 사진 대신 상품마다 일관된 아이콘+그라디언트로 그린다(오프라인 안전).
export default function ProductThumb({
  product,
  className = "",
  size = "text-5xl",
}: {
  product: { product_id: string; name: string };
  className?: string;
  size?: string;
}) {
  const { glyph, gradient } = productVisual(product);
  return (
    <div
      className={`flex items-center justify-center rounded-xl bg-gradient-to-br ${gradient} ${className}`}
    >
      <span className={size} aria-hidden>
        {glyph}
      </span>
    </div>
  );
}
