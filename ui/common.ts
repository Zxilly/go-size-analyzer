import react from "@vitejs/plugin-react";
import {BuildOptions, PluginOption} from "vite";
import {codecovVitePlugin} from "@codecov/vite-plugin";
import * as path from "node:path";


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
    if (process.env.CODECOV_TOKEN === undefined) {
        console.warn("CODECOV_TOKEN is not set, codecov plugin will be disabled");
        return undefined;
    }

    return codecovVitePlugin({
        enableBundleAnalysis: true,
        bundleName: name,
        uploadToken: process.env.CODECOV_TOKEN,
        uploadOverrides: {
            sha: getSha()
        },
        debug: true,
    })
}

export function commonPlugin(): any[] {
    return [
        react(),
    ]
}

export function build(dir: string): BuildOptions {
    return {
        outDir: path.join("dist", dir),
        minify: "terser",
        terserOptions: {
            compress: {
                passes: 2,
                dead_code: true,
            },
        },
    }
}
