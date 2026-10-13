import js from "@eslint/js";
import reactHooks from "eslint-plugin-react-hooks";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["node_modules/**", ".expo/**", "dist/**", "coverage/**", "*.config.js", "*.config.mjs"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    plugins: { "react-hooks": reactHooks },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/consistent-type-imports": ["error", { prefer: "type-imports", fixStyle: "inline-type-imports" }],
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
      "no-console": ["error", { allow: ["warn", "error"] }]
    }
  },
  {
    // Node scripts (E2E, fixtures) run outside the app bundle.
    files: ["e2e/**/*.mjs", "scripts/**/*.js"],
    languageOptions: {
      globals: {
        process: "readonly",
        console: "readonly",
        fetch: "readonly",
        FormData: "readonly",
        Blob: "readonly",
        Buffer: "readonly",
        require: "readonly",
        module: "readonly",
        __dirname: "readonly"
      }
    },
    rules: { "no-console": "off", "@typescript-eslint/no-require-imports": "off" }
  },
  {
    files: ["**/*.test.ts", "**/*.test.tsx", "jest.setup.tsx"],
    rules: { "@typescript-eslint/no-require-imports": "off" }
  }
);
