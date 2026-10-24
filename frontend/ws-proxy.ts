import type { ProxyOptions } from "vite";

export const websocketProxy: ProxyOptions = {
  target: "ws://localhost:8080",
  ws: true,
  changeOrigin: false,
  rewriteWsOrigin: false,
};
