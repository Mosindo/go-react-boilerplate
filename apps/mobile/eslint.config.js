const expoConfig = require("eslint-config-expo/flat");

module.exports = [
  ...expoConfig,
  {
    // tamagui.config.ts and src/theme are unused leftovers of the old UI kit (pending deletion).
    ignores: ["tamagui.config.ts", "src/theme/**", "dist/**", "node_modules/**", ".expo/**", "test-results/**", "playwright-report/**"]
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: { "@typescript-eslint/no-explicit-any": "error" }
  },
  {
    rules: { "no-console": ["error", { allow: ["warn", "error"] }] }
  },
  {
    // Node scripts and test runners may log and use CommonJS.
    files: ["scripts/**/*.js", "e2e/**/*.js", "e2e/**/*.ts", "playwright.config.ts"],
    languageOptions: {
      globals: { Buffer: "readonly", __dirname: "readonly", process: "readonly", require: "readonly", module: "writable" }
    },
    rules: { "no-console": "off", "@typescript-eslint/no-require-imports": "off" }
  }
];
