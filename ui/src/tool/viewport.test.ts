import type { Entry, EntryChildren } from "./entry.ts";
import { treemap } from "d3-hierarchy";
import { expect, it, vi } from "vitest";
import { getTestResult } from "../test/testhelper.ts";
import { PADDING, TOP_PADDING } from "./const.ts";
import { BaseImpl, createEntry, FileImpl, PackageImpl } from "./entry.ts";
import { buildHierarchy } from "./hierarchy.ts";
import { buildViewportHierarchy, pathFromHash } from "./viewport.ts";

it("keeps full partition coordinates and raw budgets when small tiles are omitted", () => {
  const entry = createEntry(getTestResult());
  const reference = treemap<Entry>().size([1440, 900]).paddingInner(PADDING).paddingTop(TOP_PADDING).round(true)(buildHierarchy(entry, null));
  const original = new Map(reference.descendants().map(node => [
    node.data.getID(),
    node,
  ]));
  const view = buildViewportHierarchy(entry, 1440, 900);
  expect(view.root.value).toBe(entry.getSize());
  expect(view.byID.size).toBeLessThan(original.size);
  expect(view.omitted.size).toBeGreaterThan(0);
  for (const node of view.root.descendants()) {
    const before = original.get(node.data.getID())!;
    expect(node.value).toBe(before.value);
    expect([node.x0, node.y0, node.x1, node.y1]).toEqual([
      before.x0,
      before.y0,
      before.x1,
      before.y1,
    ]);
    expect(node.data.getSize()).toBe(before.data.getSize());
  }
});

it("materializes functions only when their file is selected", () => {
  const first = new FileImpl({
    file_path: "first.go",
    size: 100,
    functions: [
      { name: "First", code_size: 100, addr: 1 },
    ],
  });
  const second = new FileImpl({
    file_path: "second.go",
    size: 100,
    functions: [
      { name: "Second", code_size: 100, addr: 2 },
    ],
  });
  const firstChildren = vi.spyOn(first, "getChildren");
  const secondChildren = vi.spyOn(second, "getChildren");
  const root = new TestEntry("files", 200, [first, second]);
  buildViewportHierarchy(root, 1000, 800);
  expect(firstChildren).not.toHaveBeenCalled();
  expect(secondChildren).not.toHaveBeenCalled();
  const selected = buildViewportHierarchy(root, 1000, 800, [root, first]);
  expect(selected.root.descendants().filter(node => node.data.getType() === "function")).toHaveLength(1);
  expect(secondChildren).not.toHaveBeenCalled();
});

it("does not open a subpixel package's descendants and retains the largest object", () => {
  const tiny = new PackageImpl({
    name: "tiny",
    type: "main",
    size: 1,
    files: [],
    subPackages: {},
    symbols: [],
  });
  const large = new FileImpl({ file_path: "large.go", size: 999_999 });
  const readTiny = vi.spyOn(tiny, "getChildren");
  const root = new TestEntry("root", 1_000_000, [tiny, large]);
  const view = buildViewportHierarchy(root, 1440, 900);
  expect(readTiny).not.toHaveBeenCalled();
  expect(view.byID.has(large.getID())).toBe(true);
  expect(view.byID.has(tiny.getID())).toBe(false);
  expect(view.root.value).toBe(1_000_000);
  const path = pathFromHash(root, "#root#tiny");
  expect(path).toEqual([root, tiny]);
  const zoomed = buildViewportHierarchy(root, 1440, 900, path);
  expect(zoomed.byID.has(tiny.getID())).toBe(true);
  expect(root.getSize()).toBe(1_000_000);
});

it("handles empty viewports, zero-sized entries, and deep hash paths without invalid rectangles", () => {
  const zero = new TestEntry("zero", 0, [
    new FileImpl({ file_path: "empty", size: 0 }),
  ]);
  const empty = buildViewportHierarchy(zero, 0, 0);
  expect(empty.root.value).toBe(0);
  expect([
    empty.root.x0,
    empty.root.x1,
    empty.root.y0,
    empty.root.y1,
  ]).toEqual([0, 0, 0, 0]);
  const path: Entry[] = [
    new FileImpl({ file_path: "deep.go", size: 1 }),
  ];
  for (let i = 0; i < 500; i++) {
    path.unshift(new TestEntry(`level-${i}`, 1, [path[0]]));
  }
  const hash = `#${path.map(item => item.getURLSafeName()).join("#")}`;
  expect(pathFromHash(path[0], hash)).toEqual(path);
  const deep = buildViewportHierarchy(path[0], 1440, 900, path);
  const leaf = deep.root.descendants().find(node => node.data === path[path.length - 1])!;
  expect(leaf.y1 - leaf.y0).toBeGreaterThan(500);
  expect(deep.root.descendants().every(node => [node.x0, node.y0, node.x1, node.y1].every(Number.isFinite))).toBe(true);
});

class TestEntry extends BaseImpl implements Entry {
  constructor(private name: string, private size: number, private children: Entry[]) { super(); }
  getType() { return "package" as const; }
  getSize() { return this.size; }
  override getName() { return this.name; }
  getChildren() { return this.children as EntryChildren["package"]; }
  toString() { return this.name; }
}
