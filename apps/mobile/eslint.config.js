const expoConfig = require("eslint-config-expo/flat");

module.exports = [
  ...expoConfig,
  { ignores: ["node_modules/", "dist/", ".expo/", "coverage/"] },
  {
    files: ["**/*.{ts,tsx}"],
    rules: {
      "react-hooks/exhaustive-deps": "error",
      "@typescript-eslint/no-explicit-any": "error"
    }
  }
];
