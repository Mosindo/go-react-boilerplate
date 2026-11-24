// https://docs.expo.dev/guides/using-eslint/
const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expoConfig,
  {
    ignores: ["dist/*", ".expo/*", "node_modules/*", "e2e/web/test-results/*"]
  },
  {
    rules: { "no-console": ["warn", { allow: ["warn", "error"] }] }
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: { "@typescript-eslint/no-explicit-any": "error" }
  },
  {
    files: ["scripts/**/*.js", "e2e/**/*.{js,mjs}"],
    rules: { "no-console": "off" }
  }
]);
