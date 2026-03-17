import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [tailwindcss(), react()],
  server: {
    host: "0.0.0.0",
    port: 5173
  },
  build: {
    rolldownOptions: {
      onLog(level, log, defaultHandler) {
        if (log.code === "EVAL") return;
        defaultHandler(level, log);
      },
    },
  },
});
