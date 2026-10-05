import path from "node:path";
import { defineConfig } from "vitest/config";

// Unit tests cover pure logic (API client, realtime client, validation, cache updates). React Native and Expo
// modules are replaced by tiny stubs; UI behaviour is covered by the Playwright suite.
export default defineConfig({
  define: { __DEV__: "true" },
  resolve: {
    alias: {
      "react-native": path.resolve(import.meta.dirname, "test/stubs/react-native.ts"),
      "expo-constants": path.resolve(import.meta.dirname, "test/stubs/expo-constants.ts"),
      "expo-secure-store": path.resolve(import.meta.dirname, "test/stubs/expo-secure-store.ts")
    }
  },
  test: { environment: "node", include: ["src/**/*.test.ts"] }
});
