import react from "@vitejs/plugin-react";
import {BuildOptions, PluginOption} from "vite";
import {codecovVitePlugin} from "@codecov/vite-plugin";


export function getSha(): string | undefined {
    const envs = process.env;
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

export function codecov(name: string): PluginOption {
    return codecovVitePlugin({
        enableBundleAnalysis: process.env.CODECOV_TOKEN !== undefined,
        bundleName: name,
        uploadToken: process.env.CODECOV_TOKEN,
        uploadOverrides: {
            sha: getSha()
        },
        debug: true,
    })
}

export function commonPlugin(): PluginOption[][] {
    return [
        react({
            babel: {
                plugins: ["babel-plugin-react-compiler"]
            }
        }),
    ]
}

export function build(): BuildOptions {
    return {
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
}