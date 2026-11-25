// https://docs.expo.dev/guides/using-eslint/
const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");
const globals = require("globals");

module.exports = defineConfig([
  expoConfig,
  {
    ignores: ["dist/*", ".expo/*", ".expo-audit/*", "web-build/*", "node_modules/*", "e2e/web/.output/*"]
  },
  {
    rules: {
      "no-console": ["warn", { allow: ["warn", "error"] }]
    }
  },
  {
    files: ["scripts/**/*.js", "e2e/**/*.js", "*.config.js"],
    languageOptions: { globals: globals.node },
    rules: { "no-console": "off" }
  }
]);
