import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { resolve } from "node:path";

const apiOrigin = "http://127.0.0.1:7332";
const devOrigins = new Set([
  "http://127.0.0.1:7331",
  "http://localhost:7331",
  "http://[::1]:7331",
]);

export default defineConfig({
  plugins: [react()],
  server: {
    port: 7331,
    strictPort: true,
    proxy: {
      "/prx.v1": {
        target: apiOrigin,
        changeOrigin: true,
        configure(proxy) {
          proxy.on("proxyReq", (proxyRequest, request) => {
            const origin = request.headers.origin;
            // Preserve the API's origin check for every caller except the
            // known local Vite development origins.
            if (origin && devOrigins.has(origin))
              proxyRequest.setHeader("origin", apiOrigin);
          });
        },
      },
    },
  },
  build: {
    outDir: resolve(__dirname, "../internal/webui/dist"),
    emptyOutDir: false,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("@xyflow")) return "graph";
          if (id.includes("@tanstack")) return "tanstack";
          if (id.includes("@bufbuild") || id.includes("@connectrpc"))
            return "rpc";
          if (
            id.includes("/node_modules/react/") ||
            id.includes("/node_modules/react-dom/")
          )
            return "react";
        },
      },
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: "./tests/setup.ts",
    exclude: ["tests/e2e/**", "node_modules/**"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/gen/**", "src/**/*.d.ts"],
      thresholds: {
        autoUpdate: true,
        statements: 53.16,
        branches: 47.65,
        functions: 40,
        lines: 53.67,
      },
    },
  },
});
