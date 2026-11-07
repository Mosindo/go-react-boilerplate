const expoConfig = require("eslint-config-expo/flat");

module.exports = [
  ...expoConfig,
  { ignores: ["dist/*", ".expo/*", "node_modules/*", "e2e/*"] },
  {
    rules: {
      "react-hooks/exhaustive-deps": "error",
    },
  },
];
