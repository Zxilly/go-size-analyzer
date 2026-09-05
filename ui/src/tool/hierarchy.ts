import type { HierarchyNode } from "d3-hierarchy";
import type { Entry } from "./entry.ts";
import { hierarchy } from "d3-hierarchy";

export function buildHierarchy(entry: Entry) {
  const root = hierarchy(entry, e => e.getChildren())
    .sort((a, b) => a.data.getSize() - b.data.getSize());
  // D3's layout values are mutable; keep them separate from raw sizes shown
  // in tooltips. Shared symbols must not inflate their containing package.
  type WeightedNode = HierarchyNode<Entry> & { value: number };
  (root as WeightedNode).value = Math.max(0, entry.getSize());
  root.eachBefore((node) => {
    if (!node.children) {
      return;
    }
    // File-backed sections and headers are independently sized. At the
    // result root only package groups share the remaining byte budget.
    const isFixed = (child: HierarchyNode<Entry>) => node.data.getType() === "result"
      && (child.data.getType() === "unknown"
        || child.data.getChildren().every(entry => entry.getType() === "section"));
    let fixed = 0;
    let total = 0;
    for (const child of node.children) {
      if (isFixed(child)) {
        fixed += Math.max(0, child.data.getSize());
      }
      else {
        total += Math.max(0, child.data.getSize());
      }
    }
    const budget = Math.max(0, (node.value ?? 0) - fixed);
    const scale = total > 0 ? Math.min(1, budget / total) : 0;
    for (const child of node.children) {
      (child as WeightedNode).value = Math.max(0, child.data.getSize()) * (isFixed(child) ? 1 : scale);
    }
  });
  return root;
}
