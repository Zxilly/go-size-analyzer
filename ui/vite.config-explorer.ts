import {defineConfig} from 'vite';
import {build, codecov, commonPlugin, getVersionTag} from "./common";
import {createHtmlPlugin} from "vite-plugin-html";

const tags = [];
const versionTag = getVersionTag();
if (versionTag) {
    tags.push(versionTag);
}

if (process.env.GSA_TELEMETRY) {
    tags.push({
        tag: "script",
        attrs: {
            defer: true,
            src: "https://trail.learningman.top/script.js",
            "data-website-id": "1aab8912-b4b0-4561-a683-81a730bdb944",
        }
    })
}

export default defineConfig({
    plugins: [
        ...commonPlugin(),
        createHtmlPlugin({
            minify: true,
            entry: './src/explorer_main.tsx',
            inject: {tags}
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
})
