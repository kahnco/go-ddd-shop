/** @type {import('next').NextConfig} */
const nextConfig = {
  // 컨테이너 배포용 최소 자체완결 서버 번들(node server.js 로 기동).
  output: "standalone",
};

export default nextConfig;
