import * as process from "node:process";
import { defineConfig } from "vite";
import { createHtmlPlugin } from "vite-plugin-html";
import { build, codecov, commonPlugin, getVersionTag } from "./vite.common";

const tags = [];
const versionTag = getVersionTag();
if (versionTag) {
  tags.push(versionTag);
}

if (process.env.GSA_TELEMETRY) {
  tags.push({
    tag: "script",
    attrs: {
      "defer": true,
      "src": "https://trail.learningman.top/script.js",
      "data-website-id": "1aab8912-b4b0-4561-a683-81a730bdb944",
      "data-domains": "gsa.zxilly.dev",
      "data-exclude-search": "true",
      "data-exclude-hash": "true",
    },
  });
}

export default defineConfig({
  plugins: [
    ...commonPlugin(),
    createHtmlPlugin({
      minify: true,
      entry: "./src/explorer_main.tsx",
      inject: {
        tags,
        data: { explorerPage: true },
      },
    }),
    {
      name: "explorer-search-files",
      generateBundle() {
        this.emitFile({ type: "asset", fileName: "robots.txt", source: "User-agent: *\nAllow: /\n\nSitemap: https://gsa.zxilly.dev/sitemap.xml\n" });
        this.emitFile({ type: "asset", fileName: "sitemap.xml", source: "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\"><url><loc>https://gsa.zxilly.dev/</loc></url></urlset>\n" });
      },
    },
    codecov("gsa-explorer"),
  ],
  clearScreen: false,
  build: build("explorer"),
  server: {
    watch: {
      usePolling: true,
    },
  },
});
