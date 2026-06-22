import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// During development the frontend talks to the Gin backend on :8080.
// Vite proxies /api and /ws so the browser only ever sees same-origin URLs.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
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
