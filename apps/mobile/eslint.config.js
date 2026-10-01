const expoConfig = require("eslint-config-expo/flat");

const jestGlobals = Object.fromEntries(
  ["jest", "describe", "it", "test", "expect", "beforeEach", "afterEach", "beforeAll", "afterAll"].map((name) => [name, "readonly"])
);

module.exports = [
  ...expoConfig,
  {
    ignores: ["node_modules/**", ".expo/**", "dist/**", "coverage/**", "scripts/**", "tamagui.config.ts"]
  },
  {
    files: ["**/*.{ts,tsx}"],
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
      "no-console": ["error", { allow: ["warn", "error"] }]
    }
  },
  {
    files: ["jest.setup.js", "**/*.test.{ts,tsx}"],
    languageOptions: { globals: jestGlobals }
  }
];
