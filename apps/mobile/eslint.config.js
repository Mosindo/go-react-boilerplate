const js = require("@eslint/js");
const globals = require("globals");
const tseslint = require("typescript-eslint");
const react = require("eslint-plugin-react");
const reactHooks = require("eslint-plugin-react-hooks");

module.exports = tseslint.config(
  {
    ignores: [
      "node_modules/**",
      ".expo/**",
      ".e2e/**",
      "dist/**",
      "babel.config.js",
      "eslint.config.js",
      "tamagui.config.ts"
    ]
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    plugins: { react, "react-hooks": reactHooks },
    languageOptions: { globals: { ...globals.browser, ...globals.es2021, __DEV__: "readonly", process: "readonly" } },
    settings: { react: { version: "detect" } },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react/jsx-key": "error",
      "react/no-unescaped-entities": "off",
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
      "no-console": ["error", { allow: ["warn", "error"] }]
    }
  },
  {
    files: ["e2e/**/*.ts"],
    rules: { "no-console": "off" }
  },
  {
    files: ["scripts/**/*.js"],
    languageOptions: { globals: { ...globals.node }, sourceType: "commonjs" },
    rules: { "no-console": "off", "@typescript-eslint/no-require-imports": "off" }
  }
);
