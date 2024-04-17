import {defineConfig, PluginOption} from 'vite'
import react from '@vitejs/plugin-react-swc'
import {viteSingleFile} from "vite-plugin-singlefile"
import * as fs from "fs"

const devDataMocker: PluginOption = {
    name: 'devDataMocker',
    transformIndexHtml: async (html) => {
        if (process.env.NODE_ENV === "production") {
            return html
        }
        const data = await fs.promises.readFile(new URL("./data.json", import.meta.url), "utf-8")
        return html.replace(`"GSA_PACKAGE_DATA"`, data)
    }
}

export default defineConfig({
    plugins: [
        react(),
        viteSingleFile(
            {
                removeViteModuleLoader: true
            }
        ),
        devDataMocker,

    ],
    clearScreen: false,
    build: {
        cssMinify: "lightningcss",
    }
})
