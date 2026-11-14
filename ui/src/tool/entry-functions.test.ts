import type { FileFunction } from "../schema/schema.ts";
import { describe, expect, it } from "vitest";
import { parseResult } from "../schema/schema.ts";
import { getTestResult } from "../test/testhelper.ts";
import { FileImpl, FunctionImpl, functionPclnBytes } from "./entry.ts";
import { buildHierarchy } from "./hierarchy.ts";

const fn: FileFunction = {
  name: "Read",
  full_name: "example.Read",
  addr: 4096,
  code_size: 64,
  pcln_bytes: 16,
  referenced_bytes: 4096,
};

describe("function entries", () => {
  it("preserves older file-only reports", () => {
    const file = new FileImpl({ file_path: "source.go", size: 80, pcln_size: 16 });
    expect(file.getChildren()).toEqual([]);
    expect(file.getSize()).toBe(80);
  });

  it("keeps reference bytes out of function and file areas", () => {
    const file = new FileImpl({
      file_path: "source.go",
      functions: [fn],
    });
    const child = file.getChildren()[0];
    expect(child.getType()).toBe("function");
    expect(child.getSize()).toBe(80);
    expect(file.getSize()).toBe(80);
    expect(buildHierarchy(file).value).toBe(80);
    expect(child.toString()).toContain("may be shared");
  });

  it("creates function nodes only for the expanded file", () => {
    const file = new FileImpl({
      file_path: "source.go",
      size: 80,
      pcln_size: 16,
      functions: [fn],
    });
    expect(buildHierarchy(file, null).descendants()).toHaveLength(1);
    expect(buildHierarchy(file, file.getID()).descendants()).toHaveLength(2);
    expect(file.getChildren()[0]).toBe(file.getChildren()[0]);
  });

  it("distinguishes unmeasured references from a measured zero", () => {
    expect(new FunctionImpl({ ...fn, referenced_bytes: undefined }, "source.go").toString()).toContain("Not measured");
    const measured = new FunctionImpl({ ...fn, referenced_bytes: 0 }, "source.go").toString();
    expect(measured).not.toContain("Not measured");
    expect(measured).toMatch(/Referenced data:\s+0 B/);
  });

  it("uses exact pcln bytes, including zero, and supports legacy components", () => {
    expect(functionPclnBytes({
      ...fn,
      pcln_bytes: 0,
      pcln_size: { name: 99 },
    })).toBe(0);
    expect(functionPclnBytes({
      ...fn,
      pcln_bytes: undefined,
      pcln_size: {
        name: 4,
        ftab: 8,
        pcdata: { pc: 3 },
        aux_data: { maps: 5 },
      },
    })).toBe(20);
  });

  it("keeps evidence and full function details when parsing noncompact JSON", () => {
    const result = getTestResult();
    const parsed = parseResult(JSON.stringify({
      ...result,
      packages: {
        example: {
          name: "example",
          type: "main",
          size: 80,
          subPackages: {},
          symbols: [],
          files: [
            {
              file_path: "source.go",
              functions: [
                {
                  ...fn,
                  reference_evidence: [
                    {
                      offset: 12,
                      size: 4,
                      section: ".rodata",
                      source: "disasm",
                      role: "reference",
                      boundary: "heuristic",
                    },
                  ],
                  reference_evidence_count: 9,
                },
              ],
            },
          ],
        },
      },
    }));
    expect(parsed?.packages.example.files[0].functions?.[0].reference_evidence_count).toBe(9);
    expect(parsed?.packages.example.files[0].functions?.[0].full_name).toBe("example.Read");
  });
});
