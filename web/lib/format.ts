// 순수 포맷 헬퍼(서버/클라이언트 공용). process.env 등 서버 전용 참조가 없어야
// 클라이언트 컴포넌트에서도 안전하게 import 할 수 있다.

export function won(amount: number): string {
  return "₩" + amount.toLocaleString("ko-KR");
}

export function statusLabel(status: string): string {
  const m: Record<string, string> = {
    PLACED: "주문 접수",
    CONFIRMED: "결제 완료",
    SHIPPED: "배송 중",
    CANCELLED: "취소됨",
    RETURN_REQUESTED: "반품 요청",
    REFUNDED: "환불 완료",
  };
  return m[status] ?? status;
}
