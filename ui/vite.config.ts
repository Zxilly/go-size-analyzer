import {defineConfig, PluginOption} from 'vite'
import react from '@vitejs/plugin-react-swc'
import {viteSingleFile} from "vite-plugin-singlefile"
import * as fs from "fs"

const dataFiller: PluginOption = {
    name: 'data-filler',
    transformIndexHtml: async (html) => {
        if (process.env.NODE_ENV === "production") {
            return html
        }
        const data = await fs.promises.readFile("data.json", "utf-8")
        return html.replace(`"GSA_PACKAGE_DATA"`, data)
    }
}

export default defineConfig({
    plugins: [
        react(),
        viteSingleFile(
            {removeViteModuleLoader: true}
        ),
        dataFiller,
    ],
    clearScreen: false,
})
