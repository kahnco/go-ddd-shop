# 01. 도메인 설계 (DDD)

> 시리즈 1편 「[DDD로 쇼핑몰 도메인 설계하기](https://blog.kahnco.me/ddd-shop-domain-design)」의 산출물.
> 코드는 2편부터. 이 문서는 앞으로의 모든 구현이 따르는 도메인의 밑그림입니다.

## 유비쿼터스 언어

| 용어 | 뜻 |
|---|---|
| 상품 (Product) | 팔 수 있는 물건. 이름·가격을 가짐 |
| 주문 (Order) | 고객이 상품을 사겠다는 요청. 여러 OrderLine으로 구성 |
| 주문 항목 (OrderLine) | 주문 안의 "어떤 상품 몇 개" |
| 재고 (Stock) | 각 상품의 남은 수량 |
| 결제 (Payment) | 주문 금액을 지불하는 행위 |
| 배송 (Shipment) | 확정된 주문을 고객에게 보내는 것 |

## Bounded Context

| Context | 책임 | 핵심 애그리거트 |
|---|---|---|
| 상품 (Catalog) | 상품 등록·조회 | Product |
| 주문 (Ordering) | 주문 생성·상태 관리 | Order |
| 재고 (Inventory) | 재고 예약·차감·복원 | StockItem |
| 결제 (Payment) | 결제 처리 | Payment |
| 배송 (Shipping) | 배송 처리 | Shipment |

각 컨텍스트는 자기 도메인 안에서 완결적이며, 서로는 **도메인 이벤트로만** 협력한다.

## Order 애그리거트 — 불변식

- 주문에는 최소 하나의 OrderLine이 있어야 한다.
- 주문 총액 = 모든 OrderLine 금액의 합 (항상 일치).
- 상태 전이는 정해진 순서로만: `PLACED → PAID → CONFIRMED → SHIPPED`.
  - 취소는 `PLACED`/`PAID` 단계에서만 가능 → `CANCELLED`.
- OrderLine은 반드시 Order 루트를 통해서만 조작된다(불변식 보호).

애그리거트 = 하나의 트랜잭션 일관성 단위. 애그리거트 사이는 이벤트로 느슨하게 연결.

## 도메인 이벤트

| 이벤트 | 발행 컨텍스트 | 의미 |
|---|---|---|
| ProductRegistered | 상품 | 상품이 등록됨 |
| OrderPlaced | 주문 | 주문이 생성됨 |
| StockReserved | 재고 | 재고가 예약(확보)됨 |
| StockInsufficient | 재고 | 재고 부족 |
| StockReleased | 재고 | (보상) 예약 재고 복원 |
| PaymentCompleted | 결제 | 결제 완료 |
| PaymentFailed | 결제 | 결제 실패 |
| OrderConfirmed | 주문 | 주문 확정 |
| OrderCancelled | 주문 | 주문 취소 |
| ShipmentStarted | 배송 | 배송 시작 |

## 주문 사가 (성공/보상 흐름)

```
OrderPlaced
  → StockReserved ──(결제 성공)──→ PaymentCompleted → OrderConfirmed → ShipmentStarted
                └──(결제 실패)──→ StockReleased(보상) → OrderCancelled
  → StockInsufficient → OrderCancelled
```

- 결과적 일관성: "주문됨"과 "결제 완료" 사이 짧은 불일치 구간을 인정하고 설계.
- 보상 트랜잭션(StockReleased)은 부가 기능이 아니라 설계의 절반. 실패 경로를 처음부터 함께 설계한다.

## 앞으로의 코드 매핑 (계층/헥사고날)

- 도메인 계층: 애그리거트 · 값 객체 · 도메인 이벤트 (순수 Go, DB/프레임워크 무관)
- 애플리케이션 계층: 유스케이스(주문하기 등) · 커맨드
- 인프라 계층: DB 리포지토리 구현 · 메시지 브로커(NATS)
- bounded context: 처음엔 패키지, 4편에서 독립 서비스로 분리
