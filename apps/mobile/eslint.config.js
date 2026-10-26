const expoConfig = require("eslint-config-expo/flat");

module.exports = [
  ...expoConfig,
  { ignores: ["dist/*", ".expo/*", "node_modules/*"] },
  {
    files: ["**/*.{ts,tsx}"],
    rules: {
      "no-console": ["warn", { allow: ["warn", "error"] }],
      "@typescript-eslint/no-explicit-any": "error"
    }
  }
];
