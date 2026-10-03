import { resolve } from "node:path";
import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    alias: {
      // The HTTP client only needs Platform; the rest of React Native cannot load under Node.
      // Tests are run from apps/mobile (npm test), so cwd-relative resolution is stable.
      "react-native": resolve(process.cwd(), "test/react-native-stub.ts")
    }
  },
  define: { __DEV__: "false" },
  test: { include: ["src/**/*.test.ts"], environment: "node" }
});
