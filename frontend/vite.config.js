import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// During development the frontend talks to the Gin backend on :8080.
// Vite proxies /api and /ws so the browser only ever sees same-origin URLs.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // Local dev: if 5173 is taken, fail loudly instead of silently hopping to
    // another port (which breaks the fixed backend CORS/proxy assumptions).
    strictPort: true,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
        changeOrigin: true,
      },
    },
  },
});
