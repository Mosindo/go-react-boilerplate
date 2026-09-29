const js = require("@eslint/js");
const globals = require("globals");
const reactHooks = require("eslint-plugin-react-hooks");
const tseslint = require("typescript-eslint");

module.exports = tseslint.config(
  { ignores: ["node_modules/**", ".expo/**", "dist/**", "web-build/**", "coverage/**"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  reactHooks.configs.flat.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      globals: { ...globals.es2022, __DEV__: "readonly" }
    },
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }]
    }
  },
  {
    files: ["**/*.js", "scripts/**/*.js", "*.config.js"],
    languageOptions: {
      sourceType: "commonjs",
      globals: { ...globals.node, ...globals.es2022 }
    },
    rules: {
      "@typescript-eslint/no-require-imports": "off"
    }
  }
);
