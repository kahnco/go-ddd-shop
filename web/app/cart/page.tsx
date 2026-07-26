import Link from "next/link";
import { getSession } from "@/lib/session";
import { getCart, listProducts } from "@/lib/api";
import CartActions, { type Row } from "./CartActions";

export default async function CartPage() {
  const session = await getSession();

  if (!session) {
    return (
      <div className="mx-auto max-w-sm text-center text-neutral-400">
        <h1 className="mb-4 text-2xl font-bold text-white">장바구니</h1>
        <p>장바구니를 보려면 로그인이 필요합니다.</p>
        <Link
          href="/login"
          className="mt-6 inline-block rounded-lg bg-blue-600 px-5 py-2 font-medium text-white hover:bg-blue-500"
        >
          로그인하기
        </Link>
      </div>
    );
  }

  // 장바구니는 product_id·수량만 갖고 있으니, 카탈로그로 이름·가격을 채운다.
  const [cart, products] = await Promise.all([
    getCart(session.customerId, session.token),
    listProducts(),
  ]);
  const priceMap = new Map(products.map((p) => [p.product_id, p]));

  const rows: Row[] = cart.items.map((it) => {
    const p = priceMap.get(it.product_id);
    return {
      product_id: it.product_id,
      name: p?.name ?? it.product_id,
      price: p?.price ?? 0,
      quantity: it.quantity,
    };
  });
  const total = rows.reduce((sum, r) => sum + r.price * r.quantity, 0);

  return (
    <div>
      <h1 className="mb-8 text-2xl font-bold">장바구니</h1>
      {rows.length === 0 ? (
        <div className="rounded-xl border border-white/10 bg-white/[0.03] p-10 text-center text-neutral-500">
          장바구니가 비어 있습니다.
          <br />
          <Link href="/" className="mt-3 inline-block text-blue-400 hover:underline">
            상품 보러 가기 →
          </Link>
        </div>
      ) : (
        <CartActions rows={rows} total={total} />
      )}
    </div>
  );
}
