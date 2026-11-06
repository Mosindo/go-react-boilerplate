const js = require("@eslint/js");
const tseslint = require("typescript-eslint");
const reactHooks = require("eslint-plugin-react-hooks");
const globals = require("globals");

module.exports = tseslint.config(
  {
    ignores: ["node_modules/**", "dist/**", ".expo/**", "test-results/**", "playwright-report/**"]
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    plugins: { "react-hooks": reactHooks },
    languageOptions: { globals: { ...globals.browser, ...globals.node, __DEV__: "readonly" } },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/consistent-type-imports": "error",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }
      ]
    }
  },
  {
    files: ["**/*.js"],
    languageOptions: { globals: globals.node },
    rules: { "@typescript-eslint/no-require-imports": "off" }
  }
);
