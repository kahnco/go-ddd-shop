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
- **개발 방식**: TDD(테스트 주도 개발) — 실패 테스트를 먼저, 도메인 규칙을 실행 가능한 명세로
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

## 실행해보기 (part-4 기준)

전체 테스트(임베디드 NATS로 이벤트 흐름까지 검증):

```bash
go test ./...
```

이벤트로 이어진 두 서비스를 직접 띄워보기 — NATS 브로커가 필요합니다.

```bash
# 1) NATS 브로커
docker run -p 4222:4222 nats:latest        # 또는: go run github.com/nats-io/nats-server/v2 -p 4222

# 2) 재고 소비자 (order.placed 구독)
NATS_URL=nats://localhost:4222 go run ./cmd/inventory

# 3) 주문 서비스 (NATS_URL 이 있으면 이벤트를 브로커로 발행)
NATS_URL=nats://localhost:4222 go run ./cmd/ordering

# 4) 주문을 넣으면, 재고 서비스가 이벤트를 받아 재고를 예약한다
curl -X POST localhost:8080/orders \
  -d '{"customer_id":"c1","items":[{"product_id":"prod-A","quantity":2,"unit_price":1000}]}'
```

> `NATS_URL` 없이 `go run ./cmd/ordering` 만 띄우면, 발행 어댑터가 로그 발행으로 대체돼
> 브로커 없이도 단독 실행됩니다(포트/어댑터 교체의 이점).

## 쿠버네티스(kind)에 배포 (part-6 기준)

```bash
# 1) 로컬 클러스터 (80 포트를 호스트로 노출)
kind create cluster --name shop --config deploy/kind/cluster.yaml

# 2) 이미지 빌드 후 kind 로 로드
docker build --build-arg SERVICE=ordering  -t go-ddd-shop/ordering:part-6  .
docker build --build-arg SERVICE=inventory -t go-ddd-shop/inventory:part-6 .
kind load docker-image go-ddd-shop/ordering:part-6 go-ddd-shop/inventory:part-6 --name shop

# 3) Ingress 컨트롤러 + 앱 매니페스트
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.3/deploy/static/provider/kind/deploy.yaml
kubectl wait -n ingress-nginx --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller --timeout=120s
kubectl apply -f deploy/k8s/

# 4) Ingress 로 주문 넣기
curl -X POST http://localhost/orders \
  -d '{"customer_id":"c1","items":[{"product_id":"prod-A","quantity":2,"unit_price":1000}]}'
kubectl logs -n shop deploy/inventory   # 재고 서비스가 이벤트를 소비한 로그

# 정리
kind delete cluster --name shop
```

> ⚠️ ordering 은 replica 2개인데 저장소가 인메모리라, POST 한 주문을 GET 하면 다른 파드로
> 라우팅돼 404 가 섞일 수 있습니다. 상태를 파드 밖(공유 저장소)으로 빼는 건 7편에서 다룹니다.

## 라이선스

MIT
