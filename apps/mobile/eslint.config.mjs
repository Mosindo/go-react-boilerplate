import js from "@eslint/js";
import globals from "globals";
import react from "eslint-plugin-react";
import reactHooks from "eslint-plugin-react-hooks";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["node_modules/**", ".expo/**", "dist/**", "scripts/**", "babel.config.js"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["e2e/**/*.mjs", "eslint.config.mjs"],
    languageOptions: { globals: { ...globals.node } },
    rules: { "no-console": "off" }
  },
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: { globals: { ...globals.browser, ...globals.node, __DEV__: "readonly" } },
    plugins: { react, "react-hooks": reactHooks },
    settings: { react: { version: "19" } },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react/jsx-key": "error",
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/consistent-type-imports": ["error", { fixStyle: "inline-type-imports", prefer: "type-imports" }],
      "no-console": "error"
    }
  }
);
