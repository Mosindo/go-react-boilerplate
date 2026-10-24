const expoConfig = require("eslint-config-expo/flat");
const tsPlugin = require("@typescript-eslint/eslint-plugin");

module.exports = [
  ...expoConfig,
  { ignores: ["dist/*", ".expo/*", "node_modules/*", "test-results/*", "playwright-report/*"] },
  {
    plugins: { "@typescript-eslint": tsPlugin },
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "react-hooks/exhaustive-deps": "error",
    },
  },
  {
    files: ["scripts/**/*.js", "*.config.js", "babel.config.js"],
    languageOptions: {
      sourceType: "commonjs",
      globals: {
        __dirname: "readonly",
        require: "readonly",
        module: "readonly",
        process: "readonly",
        console: "readonly",
      },
    },
  },
  {
    // vi.mock() must run before the module under test is imported.
    files: ["**/*.test.ts"],
    rules: { "import/first": "off" },
  },
];
