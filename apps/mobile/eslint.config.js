const { defineConfig, globalIgnores } = require("eslint/config");
const expo = require("eslint-config-expo/flat");

module.exports = defineConfig([
  globalIgnores(["node_modules/", ".expo/", "dist/", "coverage/"]),
  expo,
  {
    files: ["**/*.{ts,tsx}"],
    rules: {
      "no-console": "error",
      "@typescript-eslint/no-explicit-any": "error"
    }
  },
  {
    files: ["scripts/**/*.js", "*.config.js"],
    rules: { "no-console": "off" }
  }
]);
