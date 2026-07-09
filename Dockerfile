# syntax=docker/dockerfile:1

# ── 1단계: 빌드 ────────────────────────────────────────────────────────────
# 무거운 Go 툴체인은 빌드 단계에만 있고, 최종 이미지에는 남지 않는다(멀티스테이지).
ARG GO_VERSION=1.26
FROM golang:${GO_VERSION} AS build
WORKDIR /src

# 의존성만 먼저 받아 레이어 캐시를 살린다. 소스만 바뀌면 이 레이어는 재사용된다.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# 소스 복사 후, 어떤 서비스를 빌드할지 ARG 로 고른다(cmd/ordering, cmd/inventory).
COPY . .
ARG SERVICE
# CGO 끄고 정적 링크 → distroless/scratch 에서도 도는 단일 바이너리.
# -ldflags "-s -w" 로 디버그 심볼을 떼 이미지를 더 줄인다.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${SERVICE}

# ── 2단계: 런타임 ──────────────────────────────────────────────────────────
# distroless static: 셸도 패키지 매니저도 없는 최소 런타임. 공격 표면이 작다.
# nonroot 태그라 기본 사용자가 비루트 → 컨테이너 보안 기본기.
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
