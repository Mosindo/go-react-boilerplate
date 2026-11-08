const expoConfig = require("eslint-config-expo/flat");

const nodeGlobals = {
  Buffer: "readonly",
  process: "readonly",
  require: "readonly",
  module: "writable",
  __dirname: "readonly",
  console: "readonly",
  fetch: "readonly",
  FormData: "readonly",
  Blob: "readonly",
  setTimeout: "readonly",
  localStorage: "readonly"
};

module.exports = [
  ...expoConfig,
  { ignores: ["dist/*", ".expo/*", "node_modules/*"] },
  {
    rules: {
      "no-console": ["error", { allow: ["warn", "error"] }]
    }
  },
  {
    // Node-run end-to-end scripts: console output is their purpose.
    files: ["scripts/**/*.js"],
    languageOptions: { sourceType: "commonjs", globals: nodeGlobals },
    rules: { "no-console": "off", "@typescript-eslint/no-require-imports": "off" }
  }
];
