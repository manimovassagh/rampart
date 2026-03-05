import { defineConfig } from "tsup";

export default defineConfig({
  entry: [
    "src/index.ts",
    "src/middleware.ts",
    "src/server.ts",
    "src/client.ts",
  ],
  format: ["esm", "cjs"],
  dts: true,
  clean: true,
  external: ["next", "react", "react-dom", "@rampart/web", "@rampart/react"],
});
