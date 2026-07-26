// 서버(Next.js) → Go 서비스 호출용 베이스 URL.
// 컨테이너 안에서 compose 내부 DNS(http://catalog:8080 …)로 서버-투-서버 호출한다.
// 브라우저는 이 URL 을 절대 보지 못한다(모두 서버 컴포넌트/Route Handler 안에서만 쓰임).
// 기본값은 compose 가 호스트로 노출한 포트(로컬 `npm run dev` 로 붙을 때용).
// 컨테이너 안에서는 env(CATALOG_URL 등)로 내부 DNS(http://catalog:8080)가 주입된다.
export const services = {
  catalog: process.env.CATALOG_URL ?? "http://localhost:8084",
  customer: process.env.CUSTOMER_URL ?? "http://localhost:8085",
  cart: process.env.CART_URL ?? "http://localhost:8086",
  ordering: process.env.ORDERING_URL ?? "http://localhost:8080",
  readmodel: process.env.READMODEL_URL ?? "http://localhost:8087",
};

// 포맷 헬퍼는 클라이언트 컴포넌트도 쓰므로 순수 모듈에서 재노출.
export { won, statusLabel } from "./format";

export type Product = {
  product_id: string;
  name: string;
  price: number;
};

/** 상품 목록(카탈로그). 서버 컴포넌트에서 호출 → SSR. */
export async function listProducts(): Promise<Product[]> {
  try {
    const res = await fetch(`${services.catalog}/products`, { cache: "no-store" });
    if (!res.ok) return [];
    return (await res.json()) as Product[];
  } catch {
    return [];
  }
}

/** 상품 상세. 없으면 null. */
export async function getProduct(id: string): Promise<Product | null> {
  try {
    const res = await fetch(`${services.catalog}/products/${encodeURIComponent(id)}`, {
      cache: "no-store",
    });
    if (!res.ok) return null;
    return (await res.json()) as Product;
  } catch {
    return null;
  }
}

export type CartItem = { product_id: string; quantity: number };
export type Cart = { customer_id: string; items: CartItem[] };

/** 장바구니 조회(로그인 필요 — Bearer 토큰). */
export async function getCart(customerId: string, token: string): Promise<Cart> {
  try {
    const res = await fetch(`${services.cart}/carts/${encodeURIComponent(customerId)}`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
    if (!res.ok) return { customer_id: customerId, items: [] };
    return (await res.json()) as Cart;
  } catch {
    return { customer_id: customerId, items: [] };
  }
}

export type OrderView = {
  order_id: string;
  customer_id: string;
  status: string;
  total: number;
  channel?: string;
  items: CartItem[];
};

/** 내 주문 목록(읽기 모델 — CQRS). */
export async function getMyOrders(customerId: string): Promise<OrderView[]> {
  try {
    const res = await fetch(
      `${services.readmodel}/customers/${encodeURIComponent(customerId)}/orders`,
      { cache: "no-store" },
    );
    if (!res.ok) return [];
    return (await res.json()) as OrderView[];
  } catch {
    return [];
  }
}
