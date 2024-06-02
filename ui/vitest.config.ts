import {defineConfig} from "vitest/config";
import react from "@vitejs/plugin-react-swc";

export default defineConfig({
    plugins: [
        react(),
    ],
    test: {
        environment: 'jsdom',
        setupFiles: ['./vitest.setup.ts'],
        coverage: {
            provider: "istanbul",
            enabled: true,
            exclude: [
                "node_modules",
                "dist",
                "coverage",
                ".eslintrc.cjs",
                "vite.config.ts",
                "vite.config-explorer.ts",
                "common.ts",
                "src/tool/wasm_exec.js",
                "src/schema/schema.ts",
            ],
        },
        reporters: ["junit", "default", "github-actions"],
        outputFile: {
            "junit": "test-results.xml",
        }
    }
})