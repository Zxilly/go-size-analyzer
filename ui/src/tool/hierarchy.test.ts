import type { Package } from "../schema/schema.ts";
import { expect, it } from "vitest";
import { getTestResult } from "../test/testhelper.ts";
import { createEntry, PackageImpl } from "./entry.ts";
import { buildHierarchy } from "./hierarchy.ts";

it("keeps overlapping symbol areas within the package's byte budget", () => {
  const pkg: Package = {
    name: "example",
    type: "main",
    size: 100,
    files: [],
    subPackages: {},
    symbols: [
      { name: "example.first", addr: 100, size: 80, type: "data" },
      { name: "example.second", addr: 120, size: 80, type: "data" },
    ],
  };
  const root = buildHierarchy(new PackageImpl(pkg));
  expect(root.value).toBe(100);
  expect(root.children?.map(n => n.value)).toEqual([50, 50]);
  // Tooltips retain the original symbol sizes, including shared bytes.
  expect(root.children?.map(n => n.data.getSize())).toEqual([80, 80]);
});

it("propagates a reduced budget through nested packages", () => {
  const child: Package = {
    name: "example/child",
    type: "main",
    size: 100,
    files: [],
    subPackages: {},
    symbols: [{ name: "payload", addr: 100, size: 100, type: "data" }],
  };
  const parent: Package = {
    name: "example",
    type: "main",
    size: 100,
    files: [],
    subPackages: { child },
    symbols: [{ name: "shared", addr: 100, size: 100, type: "data" }],
  };
  const root = buildHierarchy(new PackageImpl(parent));
  root.each((node) => {
    if (node.children) {
      expect(node.children.reduce((sum, c) => sum + (c.value ?? 0), 0)).toBeCloseTo(node.value ?? 0);
    }
  });
  expect(root.value).toBe(100);
});

it("reserves the exact area of debug sections and file headers", () => {
  const source = getTestResult();
  const root = buildHierarchy(createEntry({
    ...source,
    size: 250,
    packages: {
      example: { name: "example", type: "main", size: 200, files: [], subPackages: {}, symbols: [] },
    },
    sections: [
      { ...source.sections[0], name: ".debug_info", debug: true, file_size: 100, known_size: 0 },
      { ...source.sections[0], name: ".text", debug: false, file_size: 100, known_size: 100 },
    ],
  }));
  expect(root.children?.find(n => n.data.getURLSafeName() === "debug-sections")?.value).toBe(100);
  expect(root.children?.find(n => n.data.getURLSafeName() === "main-packages")?.value).toBe(100);
  expect(root.children?.find(n => n.data.getType() === "unknown")?.value).toBe(50);
  expect(root.children?.reduce((sum, n) => sum + (n.value ?? 0), 0)).toBe(250);
});
