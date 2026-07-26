# syntax=docker/dockerfile:1

# ── 1단계: 의존성 설치 ──────────────────────────────────────────────────────
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json ./
RUN npm install

# ── 2단계: 빌드 ────────────────────────────────────────────────────────────
FROM node:22-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# output: "standalone" → .next/standalone 에 자체완결 서버가 생성된다.
RUN npm run build

# ── 3단계: 런타임 ──────────────────────────────────────────────────────────
FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
# standalone 번들 + 정적 자산만 복사(무거운 node_modules 통째로 안 넣는다).
COPY --from=build /app/public ./public
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
EXPOSE 3000
CMD ["node", "server.js"]
