import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const devPort = Number(process.env.VITE_DEV_PORT ?? "5173");
const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? "http://localhost:8080";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: devPort,
    proxy: {
      "/api": {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
});
