import react from "@vitejs/plugin-react";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { createViteLicensePlugin } from "rollup-license-plugin";
import type { Plugin } from "vite";
import { defineConfig } from "vitest/config";

const apiOrigin = "http://127.0.0.1:7332";
const licenseReportPath = resolve(
  __dirname,
  "../internal/webui/dist/oss-licenses.json",
);
const packageManifest: unknown = JSON.parse(
  readFileSync(resolve(__dirname, "../package.json"), "utf8"),
);
if (
  typeof packageManifest !== "object" ||
  packageManifest === null ||
  !("version" in packageManifest) ||
  typeof packageManifest.version !== "string"
) {
  throw new Error("root package.json has no version");
}
const developmentVersion = `${packageManifest.version}-dev`;
const devOrigins = new Set([
  "http://127.0.0.1:7331",
  "http://localhost:7331",
  "http://[::1]:7331",
]);

function serveLicenseReport(): Plugin {
  return {
    name: "serve-license-report",
    apply: "serve",
    configureServer(server) {
      server.middlewares.use(
        "/oss-licenses.json",
        (_request, response, next) => {
          try {
            response.setHeader("Content-Type", "application/json");
            response.end(readFileSync(licenseReportPath));
          } catch (error) {
            next(error);
          }
        },
      );
    },
  };
}

export default defineConfig({
  plugins: [react(), createViteLicensePlugin(), serveLicenseReport()],
  define: {
    "import.meta.env.APP_VERSION": JSON.stringify(developmentVersion),
  },
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
          return undefined;
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
      reporter: ["text", "json-summary", "json"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/gen/**", "src/**/*.d.ts"],
      thresholds: {
        statements: 43.25,
        branches: 38.37,
        lines: 43.77,
      },
    },
  },
});
