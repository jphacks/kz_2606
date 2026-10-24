import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],

  // Go バイナリに埋め込むため cmd/server/dist に出す（cmd/server/embed.go の go:embed 先）
  build: {
    outDir: "../backend/cmd/server/dist",
    emptyOutDir: true,
  },

  server: {
    // Cloudflare Tunnel / LAN からスマホで開くため
    host: true,
    allowedHosts: true,
    proxy: {
      "/api": "http://localhost:8080",
      "/ws": { target: "ws://localhost:8080", ws: true },
    },
  },
});
