import process from "node:process";
import react from "@vitejs/plugin-react-swc";
import { defineConfig } from "vite";
import { coverageConfigDefaults } from "vitest/config";

const reporters = ["default", "junit"];
if (process.env.CI) {
  reporters.push("github-actions");
}

export default defineConfig({
  plugins: [
    react(),
  ],
  test: {
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    coverage: {
      provider: "v8",
      enabled: true,
      exclude: [
        "src/tool/wasm_exec.js",
        "src/schema/schema.ts",
        "src/generated/schema.ts",
        "src/testhelper.ts",
        "**/__mocks__/**",
        "**/*.js",
        "vite.*.ts",
        ...coverageConfigDefaults.exclude,
      ],
    },
    reporters,
    outputFile: {
      junit: "test-results.xml",
    },
  },
});
