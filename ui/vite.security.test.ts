import { Buffer } from "node:buffer";
import { execSync } from "node:child_process";
import { readFileSync } from "node:fs";
import process from "node:process";
import { afterEach, expect, it, vi } from "vitest";
import { getVersionTag } from "./vite.common";

vi.mock("node:child_process", async (importOriginal) => {
  const original = await importOriginal<typeof import("node:child_process")>();
  const execSync = vi.fn();
  return { ...original, execSync, default: { ...original, execSync } };
});
vi.mock("node:fs", async (importOriginal) => {
  const original = await importOriginal<typeof import("node:fs")>();
  const readFileSync = vi.fn();
  return { ...original, readFileSync, default: { ...original, readFileSync } };
});
afterEach(() => vi.restoreAllMocks());

const malicious = `";globalThis.pwned=true;// \\ </script><script id="injected">globalThis.pwned=true</script>\u2028\u2029end`;

it("git metadata remains inert JavaScript and HTML data", () => {
  vi.mocked(execSync).mockImplementation(() => Buffer.from(malicious));
  const tag = getVersionTag()!;
  document.body.innerHTML = `<script>${tag.children}</script>`;
  expect(document.querySelectorAll("script")).toHaveLength(1);
  const info = vi.fn();
  // Execute the exact generated inline script to prove strings cannot become code.
  // eslint-disable-next-line no-new-func -- execute generated script as the browser does
  new Function("console", tag.children as string)({ info });
  expect(info.mock.calls.map(call => call[0])).toEqual([
    `Branch: ${malicious}`,
    `Commit: ${malicious}`,
    `Date: ${malicious}`,
    `Message: ${malicious}`,
  ]);
  expect((globalThis as Record<string, unknown>).pwned).toBeUndefined();
});

it("the development HTML plugin safely embeds hostile report JSON", async () => {
  vi.mocked(readFileSync).mockReturnValue(JSON.stringify({ name: malicious }));
  const { default: config } = await import("./vite.config");
  const plugins = (config as { plugins: unknown[] }).plugins.flat(Infinity);
  const plugin = plugins.find((plugin: any) => plugin?.name === "vite:html") as any;
  plugin.configResolved({ base: "/", mode: "test", root: process.cwd(), build: {}, env: {} });
  const transform = plugin.transformIndexHtml;
  const result = await (typeof transform === "function" ? transform : transform.handler)("<html><head></head><body></body></html>", { path: "/", filename: "index.html" });
  const tag = result.tags.find((tag: any) => tag.attrs?.id === "data");
  document.body.innerHTML = `<script type="application/json" id="data">${tag.children}</script>`;
  expect(document.querySelectorAll("script")).toHaveLength(1);
  expect(JSON.parse(document.getElementById("data")!.textContent!)).toEqual({ name: malicious });
});
