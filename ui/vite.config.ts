import * as fs from "node:fs"
import {HtmlTagDescriptor, defineConfig} from 'vite';
import {viteSingleFile} from "vite-plugin-singlefile"
import {createHtmlPlugin} from "vite-plugin-html";
import {build, codecov, commonPlugin, getVersionTag} from "./vite.common";


const placeHolder = `"GSA_PACKAGE_DATA"`

const getPlaceHolder = (): string => {
    if (process.env.NODE_ENV === "production") {
        return placeHolder
    }

    try {
        return fs.readFileSync(
            new URL("../testdata/result.json", import.meta.url),
            "utf-8"
        )
    } catch (e) {
        console.error("Failed to load data.json, for dev you should create one with gsa", e)
        return placeHolder
    }
}

const tags: HtmlTagDescriptor[] = [
    {
        injectTo: "head",
        tag: "script",
        attrs: {
            type: "application/json",
            id: "data"
        },
        children: getPlaceHolder()
    },

]
const versionTag = getVersionTag();
if (versionTag) {
    tags.push(versionTag);
}

export default defineConfig({
    plugins: [
        ...commonPlugin(),
        createHtmlPlugin({
            minify: true,
            entry: './src/main.tsx',
            inject: {tags}
        }),
        viteSingleFile(
            {
                removeViteModuleLoader: true
            }
        ),
        codecov("gsa-ui"),
    ],
    clearScreen: false,
    build: build("webui"),
})
