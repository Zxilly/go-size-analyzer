import process from "node:process";
import react from "@vitejs/plugin-react";
import { coverageConfigDefaults, defineConfig } from "vitest/config";

const reporters = ["default", "junit"];
if (process.env.CI) {
  reporters.push("github-actions");
}

export default defineConfig({
  plugins: [
    react() as any,
  ],
  test: {
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    coverage: {
      provider: "istanbul",
      enabled: true,
      exclude: [
        "src/tool/wasm_exec.js",
        "src/schema/schema.ts",
        "src/generated/schema.ts",
        "src/test/testhelper.ts",
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
