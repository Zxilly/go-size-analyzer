import {PluginOption, defineConfig} from 'vite'
import {viteSingleFile} from "vite-plugin-singlefile"
import * as fs from "node:fs"
import {build, codecov, commonPlugin} from "./common";

const devDataMocker: PluginOption = {
    name: 'devDataMocker',
    transformIndexHtml: async (html) => {
        if (process.env.NODE_ENV === "production") {
            return html
        }
        try {
            const data = await fs.promises.readFile(
                new URL("../data.json", import.meta.url),
                "utf-8")
            return html.replace(`"GSA_PACKAGE_DATA"`, data)
        } catch (e) {
            console.error("Failed to load data.json, for dev you should create one with gsa", e)
            return html
        }
    }
}

export default defineConfig({
    plugins: [
        ...commonPlugin(),
        viteSingleFile(
            {
                removeViteModuleLoader: true
            }
        ),
        devDataMocker,
        codecov("gsa-ui"),
    ],
    clearScreen: false,
    esbuild: {
        legalComments: 'none',
    },
    build: build("webui")
})
