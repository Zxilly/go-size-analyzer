import antfu from "@antfu/eslint-config";

export default antfu({
  react: true,
  rules: {
    "no-console": "off",
  },
  stylistic: {
    indent: 2,
    quotes: "double",
    semi: true,
  },
}, {
  ignores: [
    "dist",
    "coverage",
    "src/generated/schema.ts",
    "src/tool/wasm_exec.js",
  ],
});
