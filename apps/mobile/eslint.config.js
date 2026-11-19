// https://docs.expo.dev/guides/using-eslint/
const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");
const globals = require("globals");

module.exports = defineConfig([
  expoConfig,
  {
    ignores: ["dist/*", ".expo/*", "node_modules/*", "e2e/output/*"]
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "no-console": ["warn", { allow: ["warn", "error"] }]
    }
  },
  {
    files: ["scripts/**/*.js", "e2e/**/*.js", "*.config.js"],
    languageOptions: { globals: globals.node },
    rules: { "no-console": "off" }
  }
]);
