import {defineConfig} from 'vitest/config';
import {build, codecov, commonPlugin, getVersionTag, testConfig} from "./common";
import {createHtmlPlugin} from "vite-plugin-html";

export default defineConfig({
    plugins: [
        ...commonPlugin(),
        createHtmlPlugin({
            minify: true,
            entry: './src/explorer_main.tsx',
            inject: {
                tags: [
                    getVersionTag(),
                ]
            }
        }),
        codecov("gsa-explorer")
    ],
    clearScreen: false,
    build: build("explorer"),
    server: {
        watch: {
            usePolling: true,
        },
    },
    test: testConfig(),
})
