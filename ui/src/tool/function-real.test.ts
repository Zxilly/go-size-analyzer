import { createHash } from "node:crypto";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import process from "node:process";
import { expect, it, vi } from "vitest";
import { parseResult } from "../schema/schema.ts";
import { createEntry, FileImpl, FunctionImpl } from "./entry.ts";
import { buildHierarchy } from "./hierarchy.ts";

it.runIf(Boolean(process.env.GSA_UI_REPORT))("validates function drilldown against a real analyzer report", () => {
  const input = readFileSync(process.env.GSA_UI_REPORT!, "utf8");
  const report = parseResult(input);
  expect(report).not.toBeNull();
  const entry = createEntry(report!);
  const fileChildren = vi.spyOn(FileImpl.prototype, "getChildren");
  const collapsed = buildHierarchy(entry, null);
  const files = collapsed.descendants().filter(node => node.data instanceof FileImpl && node.data.hasFunctions());
  expect(files.length).toBeGreaterThan(0);
  expect(collapsed.descendants().some(node => node.data instanceof FunctionImpl)).toBe(false);
  expect(fileChildren).not.toHaveBeenCalled();
  const file = files[0];
  const expanded = buildHierarchy(entry, file.data.getID());
  const functions = expanded.descendants().filter(node => node.data instanceof FunctionImpl);
  expect(functions).toHaveLength(file.data.getChildren().length);
  expect(expanded.value).toBe(collapsed.value);
  expect(expanded.value).toBe(report!.size);
  for (const node of functions) {
    expect(node.value ?? 0).toBeLessThanOrEqual(node.data.getSize());
    expect(node.parent?.data.getID()).toBe(file.data.getID());
  }
  const summary = {
    report: process.env.GSA_UI_REPORT,
    sha256: createHash("sha256").update(input).digest("hex"),
    file_size: report!.size,
    initial_hierarchy_nodes: collapsed.descendants().length,
    initial_function_nodes: 0,
    initial_file_children_materialized: 0,
    files_with_function_details: files.length,
    expanded_file: file.data.getName(),
    expanded_function_nodes: functions.length,
    expanded_hierarchy_nodes: expanded.descendants().length,
    functions_with_reference_evidence: functions.filter(node => node.data instanceof FunctionImpl && node.data.getData().reference_evidence?.length).length,
    unchanged_root_area: expanded.value === collapsed.value,
  };
  const output = process.env.GSA_UI_SUMMARY ?? resolve("../temp/size-iteration/ui/real-data-summary.json");
  mkdirSync(dirname(output), { recursive: true });
  writeFileSync(output, `${JSON.stringify(summary, null, 2)}\n`);
  fileChildren.mockRestore();
});
