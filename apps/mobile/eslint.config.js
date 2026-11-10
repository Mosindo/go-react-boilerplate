// https://docs.expo.dev/guides/using-eslint/
const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expoConfig,
  {
    ignores: ["dist/*", "node_modules/*", ".expo/*", "e2e/report/*", "test-results/*"]
  },
  {
    files: ["e2e/**/*.js"],
    languageOptions: {
      sourceType: "commonjs",
      globals: { __dirname: "readonly", require: "readonly", process: "readonly", module: "writable", console: "readonly" }
    }
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "no-console": ["warn", { allow: ["warn", "error"] }]
    }
  }
]);
