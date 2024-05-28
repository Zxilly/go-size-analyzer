import {BuildOptions, HtmlTagDescriptor, PluginOption} from "vite";
import {codecovVitePlugin} from "@codecov/vite-plugin";
import * as path from "node:path";
import react from "@vitejs/plugin-react-swc";

export function getSha(): string | undefined {
    const envs = process.env;
    if (!(envs?.CI)) {
        console.info("Not a CI build");
        return undefined;
    }

    if (envs.PULL_REQUEST_COMMIT_SHA) {
        console.info(`PR build detected, sha: ${envs.PULL_REQUEST_COMMIT_SHA}`)
        return envs.PULL_REQUEST_COMMIT_SHA;
    }

    console.info(`CI build detected, not a PR build`)
    return undefined;
}

export function getVersionTag():HtmlTagDescriptor {
    return {
        tag: "script",
        children: `console.info("Version: ${sha}");`,
    }
}

const sha = getSha();

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
            sha: sha
        },
        debug: true,
    })
}

export function commonPlugin(): PluginOption[][] {
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
