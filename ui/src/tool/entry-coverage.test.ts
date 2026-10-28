import type { Result, Section } from "../schema/schema.ts";
import { expect, it } from "vitest";
import { parseResult } from "../schema/schema.ts";
import { createEntry } from "./entry.ts";
import { buildHierarchy } from "./hierarchy.ts";

function section(name: string, fileSize: number, knownSize: number, debug = false): Section {
  return {
    name,
    size: fileSize,
    file_size: fileSize,
    known_size: knownSize,
    offset: 0,
    end: fileSize,
    addr: 0,
    addr_end: fileSize,
    only_in_memory: false,
    debug,
  };
}

it("keeps only unattributed metadata in section areas and shows owned metadata under packages", () => {
  const report: Result = {
    name: "physical",
    size: 250,
    coverage: { version: 2, attributed: 140, recognized: 90, unclassified: 20, shared: 0, by_source: {} },
    packages: {
      example: {
        name: "example",
        type: "main",
        size: 140,
        metadata_size: 60,
        files: [],
        subPackages: {},
        symbols: [{ name: "example.payload", addr: 100, size: 80, type: "data" }],
      },
    },
    sections: [section(".data", 100, 80), section(".symtab", 100, 60), section(".debug_info", 20, 0, true)],
  };
  const parsed = parseResult(JSON.stringify(report));
  expect(parsed?.coverage?.version).toBe(2);
  expect(parsed?.packages.example.metadata_size).toBe(60);
  const entry = createEntry(parsed!);
  const children = entry.getChildren();
  expect(children.find(c => c.getURLSafeName() === "metadata-sections")?.getSize()).toBe(40);
  expect(children.find(c => c.getURLSafeName() === "debug-sections")?.getSize()).toBe(20);
  expect(children.find(c => c.getURLSafeName() === "unk-sections")?.getSize()).toBe(20);
  expect(children.find(c => c.getType() === "unknown")?.getSize()).toBe(30);
  const pkg = children.find(c => c.getURLSafeName() === "main-packages")?.getChildren()[0];
  expect(pkg?.getChildren().find(c => c.getName() === "File metadata")?.getSize()).toBe(60);
  expect(pkg?.getChildren().some(c => c.getType() === "disasm")).toBe(false);
  expect(children.reduce((size, c) => size + c.getSize(), 0)).toBe(250);
});

it("does not reserve file-only metadata twice outside sections", () => {
  const report: Result = {
    name: "file-only-metadata",
    size: 250,
    coverage: { version: 2, attributed: 225, recognized: 25, unclassified: 0, shared: 0, by_source: {} },
    packages: {
      example: {
        name: "example",
        type: "main",
        size: 225,
        metadata_size: 25,
        files: [{ file_path: "main.go", size: 200, pcln_size: 0 }],
        subPackages: {},
        symbols: [],
      },
    },
    sections: [section(".text", 200, 200)],
  };
  const root = buildHierarchy(createEntry(report));
  expect(root.children?.find(c => c.data.getType() === "unknown")?.value).toBe(25);
  expect(root.children?.find(c => c.data.getURLSafeName() === "main-packages")?.value).toBe(225);
  expect(root.children?.reduce((size, c) => size + (c.value ?? 0), 0)).toBe(250);
});
