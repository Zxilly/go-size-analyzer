import react from "@vitejs/plugin-react-swc";
import { defineConfig } from "vite";
import { coverageConfigDefaults } from "vitest/config";

export default defineConfig({
  plugins: [
    react(),
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
        "src/testhelper.ts",
        "**/__mocks__/**",
        "**/*.js",
        "vite.*.ts",
        ...coverageConfigDefaults.exclude,
      ],
    },
    reporters: ["junit", "default", "github-actions"],
    outputFile: {
      junit: "test-results.xml",
    },
  },
});
