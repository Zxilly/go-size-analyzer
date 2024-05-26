import {defineConfig, PluginOption} from 'vite'
import {build, codecov, commonPlugin} from "./common";
import * as fs from "node:fs";

const indexHtmlTransform: PluginOption = {
    name: 'index-html-transform',
    transformIndexHtml: {
        order: "pre",
        handler: async () => {
            return await fs.promises.readFile(
                new URL("./index-explorer.html", import.meta.url),
                "utf-8")
        }
    }
}

export default defineConfig({
    plugins: [
        indexHtmlTransform,
        ...commonPlugin(),
        codecov("gsa-explorer")
    ],
    clearScreen: false,
    esbuild: {
        legalComments: 'none',
    },
    build: build("explorer"),
    server: {
        watch: {
            usePolling: true,
        },
    }
})
