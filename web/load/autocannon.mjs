// 부하 테스트 — 홈(SSR: 서버가 카탈로그를 내부 호출) 경로를 autocannon 으로 두드린다.
//   docker compose up -d && sh deploy/seed-products.sh 후:  npm run load
// 통과 기준: non-2xx·에러 0, p99 지연이 임계 이하.
import autocannon from "autocannon";

const url = process.env.LOAD_URL ?? "http://localhost:3000/";
const connections = Number(process.env.LOAD_CONNECTIONS ?? 10);
const duration = Number(process.env.LOAD_DURATION ?? 10);
const P99_BUDGET_MS = Number(process.env.LOAD_P99_MS ?? 800);

console.log(`부하: ${url}  (연결 ${connections}, ${duration}s)`);

const result = await autocannon({ url, connections, duration });

const reqPerSec = Math.round(result.requests.average);
const p50 = result.latency.p50;
const p99 = result.latency.p99;
const non2xx = result.non2xx ?? 0;
const errors = result.errors ?? 0;

console.log(`  처리량:  ${reqPerSec} req/s`);
console.log(`  지연:    p50 ${p50}ms · p99 ${p99}ms`);
console.log(`  비2xx:   ${non2xx} · 에러: ${errors}`);

const fail = [];
if (non2xx > 0) fail.push(`비2xx 응답 ${non2xx}건`);
if (errors > 0) fail.push(`에러 ${errors}건`);
if (p99 > P99_BUDGET_MS) fail.push(`p99 ${p99}ms > 예산 ${P99_BUDGET_MS}ms`);

if (fail.length) {
  console.error("❌ 부하 기준 미달: " + fail.join(", "));
  process.exit(1);
}
console.log("✅ 부하 기준 통과");
