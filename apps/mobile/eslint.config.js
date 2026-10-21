const expoConfig = require("eslint-config-expo/flat");

module.exports = [
  ...expoConfig,
  { ignores: ["node_modules/**", "dist/**", ".expo/**", "e2e/test-results/**"] },
  {
    // French copy is full of apostrophes; in React Native <Text> they are not HTML entities.
    rules: { "react/no-unescaped-entities": "off" }
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }]
    }
  }
];
