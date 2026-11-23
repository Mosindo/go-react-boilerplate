// https://docs.expo.dev/guides/using-eslint/
const { defineConfig } = require("eslint/config");
const expoConfig = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expoConfig,
  {
    ignores: ["dist/*", ".expo/*", "node_modules/*", "e2e/.artifacts/*"]
  },
  {
    rules: {
      "no-console": ["warn", { allow: ["warn", "error"] }]
    }
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: {
      "@typescript-eslint/no-explicit-any": "error"
    }
  },
  {
    files: ["scripts/**/*.js", "e2e/**/*.js", "*.config.js", "babel.config.js"],
    languageOptions: {
      globals: { __dirname: "readonly", Buffer: "readonly", process: "readonly", require: "readonly", module: "writable" }
    },
    rules: { "no-console": "off" }
  }
]);
