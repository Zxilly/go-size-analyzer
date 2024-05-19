import {defineConfig, PluginOption} from 'vite'
import react from '@vitejs/plugin-react'
import {viteSingleFile} from "vite-plugin-singlefile"
import * as fs from "fs"
import {createHtmlPlugin} from "vite-plugin-html";
import {codecovVitePlugin} from "@codecov/vite-plugin";

const devDataMocker: PluginOption = {
    name: 'devDataMocker',
    transformIndexHtml: async (html) => {
        if (process.env.NODE_ENV === "production") {
            return html
        }
        try {
            const data = await fs.promises.readFile(new URL("../data.json", import.meta.url), "utf-8")
            return html.replace(`"GSA_PACKAGE_DATA"`, data)
        } catch (e) {
            console.error("Failed to load data.json, for dev you should create one with gsa", e)
            return html
        }
    }
}

const envs = process.env;

function getSha(): string | undefined {
    if (!(envs?.CI)) {
        console.log("Not a CI build");
        return undefined;
    }

    if (envs.PULL_REQUEST_COMMIT_SHA) {
        console.log(`PR build detected, sha: ${envs.PULL_REQUEST_COMMIT_SHA}`)
        return envs.PULL_REQUEST_COMMIT_SHA;
    }

    console.log(`CI build detected, not a PR build`)
    return undefined;
}

export default defineConfig({
    plugins: [
        react({
            babel: {
                plugins: ["babel-plugin-react-compiler"]
            }
        }),
        viteSingleFile(
            {
                removeViteModuleLoader: true
            }
        ),
        devDataMocker,
        createHtmlPlugin({
            minify: true,
        }),
        codecovVitePlugin({
            enableBundleAnalysis: process.env.CODECOV_TOKEN !== undefined,
            bundleName: "gsa-ui",
            uploadToken: process.env.CODECOV_TOKEN,
            uploadOverrides: {
                sha: getSha()
            },
            debug: true,
        }),
    ],
    clearScreen: false,
    esbuild: {
        legalComments: 'none',
    },
    build: {
        cssMinify: "lightningcss",
        minify: "terser",
        terserOptions: {
            compress: {
                passes: 2,
                ecma: 2020,
                dead_code: true,
            }
        }
    }
})
