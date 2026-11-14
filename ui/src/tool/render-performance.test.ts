import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
import process from "node:process";
import { treemap } from "d3-hierarchy";
import { expect, it } from "vitest";
import { parseResult } from "../schema/schema.ts";
import { PADDING, TOP_PADDING } from "./const.ts";
import { createEntry } from "./entry.ts";
import { buildHierarchy } from "./hierarchy.ts";
import { buildViewportHierarchy } from "./viewport.ts";

it.runIf(Boolean(process.env.GSA_RENDER_REPORT))("measures a real report's initial hierarchy", () => {
  const input = readFileSync(process.env.GSA_RENDER_REPORT!, "utf8");
  const start = performance.now();
  const report = parseResult(input)!;
  const parsed = performance.now();
  const entry = createEntry(report);
  const created = performance.now();
  const mode = process.env.GSA_RENDER_MODE ?? "viewport";
  const view = mode === "viewport"
    ? buildViewportHierarchy(entry, 1440, 900)
    : null;
  const root = view?.root ?? buildHierarchy(entry, null);
  const built = performance.now();
  const layout = view?.root ?? treemap<typeof entry>().size([1440, 900]).paddingInner(PADDING).paddingTop(TOP_PADDING).round(true)(root);
  const laidOut = performance.now();
  const nodes = layout.descendants();
  expect(root.value).toBe(report.size);
  const output = process.env.GSA_RENDER_SUMMARY!;
  mkdirSync(dirname(output), { recursive: true });
  writeFileSync(output, JSON.stringify({
    report: process.env.GSA_RENDER_REPORT,
    mode,
    viewport: [1440, 900],
    parse_ms: parsed - start,
    create_entries_ms: created - parsed,
    hierarchy_ms: built - created,
    layout_ms: laidOut - built,
    hierarchy_nodes: nodes.length,
    created_hierarchy_nodes: view?.createdNodes ?? nodes.length,
    renderable_nodes: nodes.filter(n => n.x1 - n.x0 >= 2 && n.y1 - n.y0 >= 2).length,
    root_bytes: root.value,
    heap_used_bytes: process.memoryUsage().heapUsed,
  }, null, 2));
});
