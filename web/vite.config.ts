import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { resolve } from "node:path";

export default defineConfig({
  plugins: [react()],
  server: { port: 5173, proxy: { "/prmap.v1": "http://127.0.0.1:7331" } },
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
          if (id.includes("node_modules/react")) return "react";
        },
      },
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: "./tests/setup.ts",
    exclude: ["tests/e2e/**", "node_modules/**"],
    coverage: { provider: "v8", reporter: ["text", "json-summary"] },
  },
});
