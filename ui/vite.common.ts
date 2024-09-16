import type { BuildOptions, HtmlTagDescriptor, Plugin, PluginOption } from "vite";
import { execSync } from "node:child_process";
import * as path from "node:path";
import process from "node:process";
import { codecovVitePlugin } from "@codecov/vite-plugin";
import react from "@vitejs/plugin-react-swc";

export function getSha(): string | undefined {
  const envs = process.env;
  if (!(envs?.CI)) {
    console.info("Not a CI build");
    return undefined;
  }

  if (envs.PULL_REQUEST_COMMIT_SHA) {
    console.info(`PR build detected, sha: ${envs.PULL_REQUEST_COMMIT_SHA}`);
    return envs.PULL_REQUEST_COMMIT_SHA;
  }

  console.info(`CI build detected, not a PR build`);
  return envs.GITHUB_SHA;
}

export function getVersionTag(): HtmlTagDescriptor | null {
  try {
    const commitDate = execSync("git log -1 --format=%cI").toString().trimEnd();
    const branchName = execSync("git rev-parse --abbrev-ref HEAD").toString().trimEnd();
    const commitHash = execSync("git rev-parse HEAD").toString().trimEnd();
    const lastCommitMessage = execSync("git show -s --format=%s").toString().trimEnd();

    return {
      tag: "script",
      children:
                `console.info("Branch: ${branchName}");`
                + `console.info("Commit: ${commitHash}");`
                + `console.info("Date: ${commitDate}");`
                + `console.info("Message: ${lastCommitMessage}");`,
    };
  }
  catch (e) {
    console.warn("Failed to get git info", e);
    return null;
  }
}

export function codecov(name: string): Plugin[] | undefined {
  if (process.env.CODECOV_TOKEN === undefined) {
    console.info("CODECOV_TOKEN is not set, codecov plugin will be disabled");
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
  });
}

export function commonPlugin(): (PluginOption[] | Plugin | Plugin[])[] {
  return [
    react(),
  ];
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
  };
}
