const expoConfig = require("eslint-config-expo/flat");

module.exports = [
  ...expoConfig,
  {
    ignores: ["node_modules/**", ".expo/**", "dist/**", "scripts/**", "coverage/**"]
  },
  {
    files: ["**/*.ts", "**/*.tsx"],
    rules: {
      "no-console": ["error", { allow: ["warn", "error"] }],
      "@typescript-eslint/no-explicit-any": "error",
      "react-hooks/exhaustive-deps": "error"
    }
  }
];
