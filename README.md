# go-ddd-shop

**Go로 만드는 이벤트 기반 쇼핑몰 — DDD 설계부터 쿠버네티스 배포까지.**

이 저장소는 [Kahnco 블로그](https://blog.kahnco.me)의 실전 따라하기 시리즈에서 함께 만들어가는 코드입니다.
빈 폴더에서 시작해, 도메인 설계(DDD) → Go 구현 → 이벤트 기반 아키텍처(EDD) → 컨테이너 → 쿠버네티스 배포까지
한 편씩 손으로 쌓아 올립니다.

## 무엇을 만드나

쇼핑몰은 도메인 주도 설계(DDD)와 이벤트 기반 설계(EDD)를 배우기에 좋은 예제입니다.
**상품 · 주문 · 재고 · 결제 · 배송** 이 자연스럽게 경계(bounded context)로 나뉘고,
"주문됨 → 재고 차감 → 결제 → 배송" 같은 흐름이 도메인 이벤트로 선명하게 이어집니다.

## 기술 스택

- **언어**: Go 1.26
- **아키텍처**: DDD(도메인 주도 설계) + EDD(이벤트 기반) + 헥사고날/계층 구조
- **메시징**: NATS (도메인 이벤트 발행/구독)
- **저장소**: PostgreSQL
- **배포**: 컨테이너 → 쿠버네티스(`kind` 로컬 클러스터)
- **CI/CD·관찰성**: GitHub Actions, GitOps, 메트릭·추적

## 시리즈 목차

1. DDD로 쇼핑몰 도메인 설계하기 — 이벤트 스토밍, bounded context, 도메인 이벤트
2. Go로 도메인 모델링 — 계층/헥사고날 구조, 애그리거트·값 객체·리포지토리 포트
3. 유스케이스와 API — 애플리케이션 계층, DB 리포지토리, 도는 단일 서비스
4. 이벤트로 컨텍스트 잇기(EDD) — 메시지 브로커, 발행/구독, 결과적 일관성·saga
5. 컨테이너에 담기 — Go 멀티스테이지 Dockerfile
6. 로컬 쿠버네티스에 배포 — kind, Deployment·Service·Ingress
7. 상태·설정·스케일링 — StatefulSet/PVC, ConfigMap/Secret, probe·HPA
8. CI/CD와 관찰성 — GitHub Actions·GitOps, 이벤트 흐름 추적

## 편별로 따라오기

각 편이 끝난 시점의 코드에 **`part-N` 태그**를 찍어 둡니다. 원하는 지점부터 따라올 수 있습니다.

```bash
git clone https://github.com/kahnco/go-ddd-shop.git
cd go-ddd-shop
git checkout part-1     # 1편이 끝난 시점의 코드
```

## 라이선스

MIT
