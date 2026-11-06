import { defineConfig } from "vite";
import path from "path";

export default defineConfig({
  build: {
    lib: {
      entry: path.resolve(__dirname, "src/index.ts"),
      name: "NotifluxBell",
      fileName: () => "notiflux-bell.js",
      formats: ["es", "umd"],
    },
    outDir: "dist",
    emptyOutDir: true,
    minify: "esbuild",
    rollupOptions: {
      external: ["react", "react-dom"],
      output: {
        globals: {
          react: "React",
          "react-dom": "ReactDOM",
        },
      },
    },
  },
});
