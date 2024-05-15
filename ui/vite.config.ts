import {defineConfig, PluginOption} from 'vite'
import react from '@vitejs/plugin-react-swc'
import {viteSingleFile} from "vite-plugin-singlefile"
import * as fs from "fs"
import {createHtmlPlugin} from "vite-plugin-html";
import {codecovVitePlugin} from "@codecov/vite-plugin";
import child_process from "child_process";

const devDataMocker: PluginOption = {
    name: 'devDataMocker',
    transformIndexHtml: async (html) => {
        if (process.env.NODE_ENV === "production") {
            return html
        }
        try {
            const data = await fs.promises.readFile(new URL("./data.json", import.meta.url), "utf-8")
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

    let commit = envs?.GITHUB_SHA;

    if (envs?.GITHUB_HEAD_REF && envs?.GITHUB_HEAD_REF !== "") {
        const prRegex = /refs\/pull\/([0-9]+)\/merge/;
        const matches = prRegex.exec(envs?.GITHUB_REF ?? "");
        if (!matches) {
            throw new Error(`Failed to get PR number from ${envs?.GITHUB_REF}`);
        }

        const mergeCommitRegex = /^[a-z0-9]{40} [a-z0-9]{40}$/;

        const mergeCommitMessage = child_process.execSync("git show --no-patch --format=%P").toString();
        if (mergeCommitMessage === "") {
            throw new Error("Failed to get merge commit message");
        }

        if (mergeCommitRegex.exec(mergeCommitMessage)) {
            commit = mergeCommitMessage.split(" ")[1];
        } else {
            const singleParentRegex = /[a-z0-9]{40}/;
            if (singleParentRegex.exec(mergeCommitMessage)) {
                commit = mergeCommitMessage;
            } else {
                throw new Error(`Failed to get commit from ${mergeCommitMessage}`);
            }
        }
    } else {
        console.log("Not a PR build");
    }

    console.log(`Reporting bundle for commit ${commit}`);
    return commit;
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
        createHtmlPlugin({
            minify: true,
        }),
        codecovVitePlugin({
            enableBundleAnalysis: process.env.CODECOV_TOKEN !== undefined,
            bundleName: "gsa-ui",
            uploadToken: process.env.CODECOV_TOKEN,
            uploadOverrides: {
                sha: getSha()
            }
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
