const { defineConfig, globalIgnores } = require("eslint/config");
const expo = require("eslint-config-expo/flat");

module.exports = defineConfig([
  expo,
  globalIgnores([
    "dist/**",
    ".expo/**",
    "node_modules/**",
    // Legacy SaaS leftovers awaiting deletion (see tsconfig.json exclude).
    "tamagui.config.ts",
    "scripts/e2e-ui-setup.js",
    "src/screens/**",
    "src/shared/**",
    "src/api/platform.ts",
    "src/api/endpoints.ts"
  ]),
  {
    files: ["scripts/**/*.js"],
    languageOptions: {
      globals: {
        Buffer: "readonly",
        process: "readonly",
        require: "readonly",
        console: "readonly",
        fetch: "readonly",
        FormData: "readonly",
        Blob: "readonly",
        __dirname: "readonly",
        module: "readonly"
      }
    }
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: {
      "@typescript-eslint/no-explicit-any": "error"
    }
  }
]);
