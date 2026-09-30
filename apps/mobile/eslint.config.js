// https://docs.expo.dev/guides/using-eslint/
const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expoConfig,
  {
    ignores: ["dist/*", ".expo/*", ".expo-audit/*", "node_modules/*", "e2e/web/test-results/*", "e2e/web/playwright-report/*"]
  }
]);
