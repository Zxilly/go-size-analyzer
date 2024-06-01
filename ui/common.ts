import {BuildOptions, HtmlTagDescriptor, PluginOption} from "vite";
import {codecovVitePlugin} from "@codecov/vite-plugin";
import * as path from "node:path";
import react from "@vitejs/plugin-react-swc";
import {execSync} from "node:child_process";
import type { InlineConfig } from 'vitest';

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
    return envs.GITHUB_SHA;
}

export function getVersionTag(): HtmlTagDescriptor {
    const commitDate = execSync('git log -1 --format=%cI').toString().trimEnd();
    const branchName = execSync('git rev-parse --abbrev-ref HEAD').toString().trimEnd();
    const commitHash = execSync('git rev-parse HEAD').toString().trimEnd();
    const lastCommitMessage = execSync('git show -s --format=%s').toString().trimEnd();

    return {
        tag: "script",
        children: `
        console.info("Branch: ${branchName}");
        console.info("Commit: ${commitHash}");
        console.info("Date: ${commitDate}");
        console.info("Message: ${lastCommitMessage}");
        `.trim(),
    }
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
            sha: getSha(),
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

export function testConfig(): InlineConfig {
    return {
        coverage: {
            provider: "v8",
            enabled: true,
            exclude: [
                "node_modules",
                "dist",
                "coverage",
                "vite.config.ts",
                "vite.config-explorer.ts",
                "common.ts",
                "src/tool/wasm_exec.js"
            ]
        }
    }
}

