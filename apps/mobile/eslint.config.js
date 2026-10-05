const js = require("@eslint/js");
const globals = require("globals");
const tseslint = require("typescript-eslint");
const reactHooks = require("eslint-plugin-react-hooks");

module.exports = tseslint.config(
  { ignores: ["dist/**", ".expo/**", "node_modules/**", "test-results/**", "playwright-report/**"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    plugins: { "react-hooks": reactHooks },
    languageOptions: { globals: { ...globals.browser, __DEV__: "readonly" } },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/consistent-type-imports": ["error", { fixStyle: "inline-type-imports" }],
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
      "no-console": ["error", { allow: ["warn", "error"] }]
    }
  },
  {
    files: ["scripts/**/*.js", "*.config.js", "babel.config.js"],
    languageOptions: { globals: { ...globals.node }, sourceType: "commonjs" },
    rules: { "@typescript-eslint/no-require-imports": "off", "no-console": "off" }
  }
);
