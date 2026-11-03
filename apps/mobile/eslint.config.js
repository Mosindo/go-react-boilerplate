const js = require("@eslint/js");
const tseslint = require("typescript-eslint");
const reactHooks = require("eslint-plugin-react-hooks");

module.exports = tseslint.config(
  { ignores: ["node_modules/**", ".expo/**", "dist/**", "scripts/**", "*.config.js", "babel.config.js"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["e2e/**/*.mjs"],
    languageOptions: {
      globals: { process: "readonly", fetch: "readonly", console: "readonly", Buffer: "readonly", localStorage: "readonly" }
    }
  },
  {
    plugins: { "react-hooks": reactHooks },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }]
    }
  }
);
