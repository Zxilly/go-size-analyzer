import type { HierarchyNode, HierarchyRectangularNode } from "d3-hierarchy";
import type { Entry } from "./entry.ts";
import { hierarchy, treemap, treemapSquarify } from "d3-hierarchy";
import { PADDING, TOP_PADDING } from "./const.ts";

// Tiny rectangles remain part of their parent's area. Zooming recalculates
// the partition; the selected parent's item browser also exposes every child.
export const MIN_TILE_AREA = 256;
const MIN_TILE_SIDE = 4;
const MIN_EXPAND_AREA = 1024;

type ViewNode = HierarchyRectangularNode<Entry> & { value: number; depth: number; height: number };

function isFixed(parent: Entry, child: Entry) {
  return parent.getType() === "result" && (child.getType() === "unknown"
    || child.getChildren().every(entry => entry.getType() === "section"));
}

function childBudgets(parent: Entry, children: Entry[], budget: number): number[] {
  let fixed = 0;
  let flexible = 0;
  for (const child of children) {
    if (isFixed(parent, child)) {
      fixed += Math.max(0, child.getSize());
    }
    else {
      flexible += Math.max(0, child.getSize());
    }
  }
  const scale = flexible > 0
    ? Math.min(1, Math.max(0, budget - fixed) / flexible)
    : 0;
  return children.map(child => Math.max(0, child.getSize()) * (isFixed(parent, child)
    ? 1
    : scale));
}

export function pathFromHash(entry: Entry, hash: string): Entry[] | null {
  const parts = hash.replace(/^#/, "").split("#");
  if (parts[0] !== entry.getURLSafeName()) {
    return null;
  }
  const path = [entry];
  for (const part of parts.slice(1)) {
    const found = path[path.length - 1].getChildren().find(child => child.getURLSafeName() === part);
    if (!found) {
      return null;
    }
    path.push(found);
  }
  return path;
}

export function buildViewportHierarchy(entry: Entry, width: number, height: number, selection: Entry[] | null = null) {
  const path = selection?.[0] === entry
    ? selection
    : [entry];
  const expandedFile = path.find(item => item.getType() === "file");
  const omitted = new Map<number, number>();
  let createdNodes = 0;

  function makeNode(data: Entry, value: number, parent: ViewNode | null): ViewNode {
    // Only immediate children enter D3. Subtrees are opened after their
    // rectangle is known, so offscreen detail never becomes a hierarchy.
    const node = hierarchy(data, () => null) as ViewNode;
    node.value = value;
    node.parent = parent;
    node.depth = parent
      ? parent.depth + 1
      : 0;
    createdNodes++;
    return node;
  }

  function expand(node: ViewNode) {
    if (node.data.getType() === "file" && node.data !== expandedFile) {
      return;
    }
    const children = node.data.getChildren();
    if (children.length === 0) {
      return;
    }
    const budgets = childBudgets(node.data, children, node.value);
    node.children = children.map((child, index) => makeNode(child, budgets[index], node))
      .sort((a, b) => a.data.getSize() - b.data.getSize());
  }

  // Compute the selected node's inherited physical-area budget by walking
  // its path, without building the rest of the hierarchy.
  let selectedBudget = Math.max(0, entry.getSize());
  for (let i = 1; i < path.length; i++) {
    const children: Entry[] = path[i - 1].getChildren();
    selectedBudget = childBudgets(path[i - 1], children, selectedBudget)[children.indexOf(path[i])];
  }
  const root = makeNode(entry, selectedBudget, null);
  let cursor = root;
  const forced = new Set<number>();
  for (const item of path.slice(1)) {
    const child = makeNode(item, selectedBudget, cursor);
    cursor.children = [child];
    forced.add(item.getID());
    cursor = child;
  }
  expand(cursor);

  const layout = treemap<Entry>()
    .size([
      Math.max(0, width),
      Math.max(0, height),
    ])
    .paddingInner(PADDING)
    .paddingTop(node => node.depth < path.length - 1
      ? Math.min(TOP_PADDING, Math.max(0, height) / (path.length * 3))
      : TOP_PADDING)
    .round(true)
    .tile((node, x0, y0, x1, y1) => {
      treemapSquarify(node, x0, y0, x1, y1);
      const children = node.children ?? [];
      const visible = children.filter(child => forced.has(child.data.getID())
        || ((child.x1 - child.x0 - PADDING) >= MIN_TILE_SIDE
          && (child.y1 - child.y0 - PADDING) >= MIN_TILE_SIDE
          && (child.x1 - child.x0 - PADDING) * (child.y1 - child.y0 - PADDING) >= MIN_TILE_AREA));
      if (visible.length !== children.length) {
        omitted.set(node.data.getID(), children.length - visible.length);
      }
      node.children = visible.length ? visible : undefined;
      for (const child of visible) {
        if (forced.has(child.data.getID())) {
          continue;
        }
        const w = child.x1 - child.x0 - PADDING;
        const h = child.y1 - child.y0 - PADDING;
        if (w >= 12 && h >= TOP_PADDING + 12 && w * h >= MIN_EXPAND_AREA) {
          expand(child as ViewNode);
        }
      }
    });
  // D3 visits children after each tile callback, including the new shallow
  // children attached above. Values are never recomputed from visible leaves.
  layout(root);
  const byID = new Map<number, HierarchyNode<Entry>>();
  root.eachAfter((node) => {
    (node as ViewNode).height = node.children
      ? 1 + node.children.reduce((height, child) => Math.max(height, child.height), 0)
      : 0;
    byID.set(node.data.getID(), node);
  });
  return { root, byID, omitted, createdNodes };
}
