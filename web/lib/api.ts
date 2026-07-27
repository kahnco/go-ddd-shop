// Go 백엔드에 대한 유일한 어댑터(포트/어댑터). 모든 HTTP 호출이 여기 모인다.
// - 서버 컴포넌트는 조회 함수를(SSR), Route Handler(BFF)는 변경 함수를 쓴다.
// - next/server·next/headers 에 의존하지 않아, 테스트에서 global fetch 만 목킹하면 된다.
export { won, statusLabel } from "./format";

// 기본값은 compose 가 호스트로 노출한 포트(로컬 `npm run dev` 용).
// 컨테이너 안에서는 env 로 내부 DNS(http://catalog:8080)가 주입된다.
export const services = {
  catalog: process.env.CATALOG_URL ?? "http://localhost:8084",
  customer: process.env.CUSTOMER_URL ?? "http://localhost:8085",
  cart: process.env.CART_URL ?? "http://localhost:8086",
  ordering: process.env.ORDERING_URL ?? "http://localhost:8080",
  readmodel: process.env.READMODEL_URL ?? "http://localhost:8087",
};

// 백엔드가 4xx/5xx 를 주면 던지는 에러. status 를 담아 라우트 핸들러가 그대로 되돌린다.
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

const jsonHeaders = { "Content-Type": "application/json" };
function bearer(token: string) {
  return { Authorization: `Bearer ${token}` };
}

// 공통 fetch: 성공 시 JSON(없으면 {}), 실패 시 ApiError.
async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  const text = await res.text();
  const body = text ? JSON.parse(text) : {};
  if (!res.ok) {
    throw new ApiError(res.status, body?.error ?? `요청 실패 (${res.status})`);
  }
  return body as T;
}

// ─── 타입 ────────────────────────────────────────────────────────────────
export type Product = { product_id: string; name: string; price: number };
export type CartItem = { product_id: string; quantity: number };
export type Cart = { customer_id: string; items: CartItem[] };
export type OrderView = {
  order_id: string;
  customer_id: string;
  status: string;
  total: number;
  channel?: string;
  items: CartItem[];
};

// ─── 조회(서버 컴포넌트, SSR) — 실패해도 화면이 죽지 않게 기본값 ────────────
export async function listProducts(): Promise<Product[]> {
  try {
    return await apiFetch<Product[]>(`${services.catalog}/products`, { cache: "no-store" });
  } catch {
    return [];
  }
}

export async function getProduct(id: string): Promise<Product | null> {
  try {
    return await apiFetch<Product>(`${services.catalog}/products/${encodeURIComponent(id)}`, {
      cache: "no-store",
    });
  } catch {
    return null;
  }
}

export async function getCart(customerId: string, token: string): Promise<Cart> {
  try {
    return await apiFetch<Cart>(`${services.cart}/carts/${encodeURIComponent(customerId)}`, {
      headers: bearer(token),
      cache: "no-store",
    });
  } catch {
    return { customer_id: customerId, items: [] };
  }
}

export async function getMyOrders(customerId: string): Promise<OrderView[]> {
  try {
    return await apiFetch<OrderView[]>(
      `${services.readmodel}/customers/${encodeURIComponent(customerId)}/orders`,
      { cache: "no-store" },
    );
  } catch {
    return [];
  }
}

// ─── 변경(Route Handler, BFF) — 실패는 ApiError 로 던진다 ──────────────────
export async function login(email: string, password: string): Promise<string> {
  const { token } = await apiFetch<{ token: string }>(`${services.customer}/auth/login`, {
    method: "POST",
    headers: jsonHeaders,
    body: JSON.stringify({ email, password }),
  });
  return token;
}

export async function register(email: string, password: string, name: string): Promise<void> {
  await apiFetch(`${services.customer}/auth/register`, {
    method: "POST",
    headers: jsonHeaders,
    body: JSON.stringify({ email, password, name }),
  });
}

export async function addCartItem(
  customerId: string,
  token: string,
  productId: string,
  quantity: number,
): Promise<void> {
  await apiFetch(`${services.cart}/carts/${encodeURIComponent(customerId)}/items`, {
    method: "POST",
    headers: { ...jsonHeaders, ...bearer(token) },
    body: JSON.stringify({ product_id: productId, quantity }),
  });
}

export async function removeCartItem(
  customerId: string,
  token: string,
  productId: string,
): Promise<void> {
  await apiFetch(
    `${services.cart}/carts/${encodeURIComponent(customerId)}/items/${encodeURIComponent(productId)}`,
    { method: "DELETE", headers: bearer(token) },
  );
}

export async function checkout(customerId: string, token: string): Promise<string> {
  const { order_id } = await apiFetch<{ order_id: string }>(
    `${services.cart}/carts/${encodeURIComponent(customerId)}/checkout`,
    { method: "POST", headers: { ...bearer(token), "X-Client-Channel": "web" } },
  );
  return order_id;
}

export async function requestReturn(orderId: string, token: string): Promise<void> {
  await apiFetch(`${services.ordering}/orders/${encodeURIComponent(orderId)}/return`, {
    method: "POST",
    headers: bearer(token),
  });
}
