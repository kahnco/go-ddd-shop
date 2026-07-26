#!/bin/sh
# 카탈로그에 데모 상품을 넣는다(카탈로그는 기동 시 비어 있으므로).
#   docker compose up -d 뒤에:  sh deploy/seed-products.sh
# 상품 등록 이벤트(catalog.product.added)가 주문 서비스의 가격 프로젝션도 갱신한다.
CATALOG=${CATALOG:-http://localhost:8084}

add() {
  curl -s -o /dev/null -w "  [%{http_code}] $1\n" \
    -X POST "$CATALOG/products" -H 'Content-Type: application/json' -d "$1"
}

add '{"product_id":"prod-A","name":"무선 이어폰","price":39000}'
add '{"product_id":"prod-B","name":"기계식 키보드","price":89000}'
add '{"product_id":"prod-C","name":"USB-C 멀티 허브","price":25000}'
add '{"product_id":"prod-D","name":"노트북 스탠드","price":32000}'
echo "완료 — http://localhost:3000 에서 확인하세요."
