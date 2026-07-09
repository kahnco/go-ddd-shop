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

### 심화: 쇼핑몰 고도화 (part-9~)

주문 여정이 이벤트만으로 끝까지 흐르도록 완성하고 리팩터링으로 다듬습니다.

1. 사가를 닫다 — 결제 서비스와 자동 취소 (주문 PLACED→PAID→CONFIRMED, 재고 부족 시 자동 취소)
2. 배송과 완전한 보상 — 배송 서비스, 결제 실패 시 재고 복원
3. 상품 카탈로그 — 진짜 상품과 가격
4. 신뢰할 수 있는 이벤트 — 아웃박스와 멱등성

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

## 쿠버네티스(kind)에 배포 (part-8 기준)

```bash
# 1) 로컬 클러스터 (80 포트를 호스트로 노출)
kind create cluster --name shop --config deploy/kind/cluster.yaml

# 2) 이미지 빌드 후 kind 로 로드 (주문·재고·결제)
docker build --build-arg SERVICE=ordering  -t go-ddd-shop/ordering:part-9  .
docker build --build-arg SERVICE=inventory -t go-ddd-shop/inventory:part-9 .
docker build --build-arg SERVICE=payment   -t go-ddd-shop/payment:part-9   .
kind load docker-image go-ddd-shop/ordering:part-9 go-ddd-shop/inventory:part-9 go-ddd-shop/payment:part-9 --name shop

# 3) Ingress 컨트롤러 + metrics-server(HPA용) + 앱 매니페스트
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.3/deploy/static/provider/kind/deploy.yaml
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl patch -n kube-system deployment metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
kubectl wait -n ingress-nginx --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller --timeout=120s
kubectl apply -f deploy/k8s/namespace.yaml && kubectl apply -f deploy/k8s/

# 4) 주문을 만들고, 같은 주문을 여러 번 조회 → 이제 replica가 Postgres를 공유해 항상 200
OID=$(curl -s -X POST http://localhost/orders \
  -d '{"customer_id":"c1","items":[{"product_id":"prod-A","quantity":2,"unit_price":1000}]}' \
  | sed -E 's/.*"order_id":"([^"]+)".*/\1/')
curl -s http://localhost/orders/$OID

# 5) 부하를 주면 HPA가 ordering 파드를 자동으로 늘린다
kubectl get hpa -n shop -w

# 정리
kind delete cluster --name shop
```

7편에서 상태를 파드 밖 **PostgreSQL(StatefulSet·PVC)** 로 빼서 6편의 404 문제를 해결했습니다.
설정은 **ConfigMap**(NATS_URL), 비밀은 **Secret**(DATABASE_URL)으로 분리하고, **probe**(/healthz·/readyz)와
**HPA**(CPU 70% 기준 2→5 파드)를 붙였습니다.

8편에서 **관찰성** 을 더했습니다 — 상관 ID(`X-Correlation-ID`)를 이벤트에 실어 주문 하나를
주문→재고 서비스에 걸쳐 같은 ID 로 추적하고, `/metrics`(Prometheus)로 요청·이벤트 수를 노출합니다.
`.github/workflows` 에 **CI/CD** 파이프라인(포맷·정적검사·테스트·이미지 빌드, 태그 시 GHCR 푸시)이
있습니다. 튜토리얼 리포라 불필요한 실행을 피하려 **수동 실행(workflow_dispatch)** 으로 두었습니다 —
실제 프로젝트에서는 주석 처리된 `push`/`pull_request` 트리거를 켜 자동화합니다.

```bash
# 관찰성 확인
curl http://localhost/metrics                          # 프로메테우스 메트릭
kubectl logs -n shop deploy/ordering  | grep correlation_id   # 주문 서비스 로그
kubectl logs -n shop deploy/inventory | grep <그 correlation_id>  # 같은 ID 가 재고 서비스에도
```

## 라이선스

MIT
