const expo = require("eslint-config-expo/flat");

module.exports = [
  ...expo,
  { ignores: ["dist/**", "node_modules/**", ".expo/**", "e2e/**"] },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
      "react-hooks/exhaustive-deps": "error"
    }
  }
];
