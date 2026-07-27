import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./test/setup.ts"],
    include: ["{app,lib}/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "v8",
      include: ["app/**", "lib/**"],
      exclude: ["**/*.test.*", "app/**/layout.tsx"],
    },
  },
  resolve: {
    // tsconfig 의 "@/*": ["./*"] 를 vitest 에서도 맞춘다.
    alias: { "@": fileURLToPath(new URL(".", import.meta.url)) },
  },
});
