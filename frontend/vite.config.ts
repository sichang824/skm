import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { visualizer } from "rollup-plugin-visualizer";

const proxyTarget = process.env.VITE_PROXY_TARGET ?? "http://localhost:8080";

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    visualizer({
      open: false,  // Don't auto-open browser
      filename: "dist/stats.html",
      gzipSize: true,
      brotliSize: true,
    }),
  ],
  server: {
    proxy: {
      "/api": proxyTarget,
      "/healthz": proxyTarget,
      "/version": proxyTarget,
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: "./vitest.setup.ts",
  },
});
