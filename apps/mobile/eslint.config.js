// https://docs.expo.dev/guides/using-eslint/
const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expoConfig,
  {
    ignores: ["dist/*", "web-build/*", "node_modules/*", ".expo/*", "playwright-report/*", "test-results/*"]
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: {
      "@typescript-eslint/no-explicit-any": "error"
    }
  },
  {
    rules: {
      "no-console": ["warn", { allow: ["warn", "error"] }]
    }
  },
  {
    files: ["scripts/**/*.js", "e2e/**/*.ts", "playwright.config.ts"],
    languageOptions: {
      globals: { Buffer: "readonly", __dirname: "readonly", process: "readonly", require: "readonly" }
    },
    rules: {
      "no-console": "off"
    }
  }
]);
