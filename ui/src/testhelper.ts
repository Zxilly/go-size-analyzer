import { readFileSync } from "node:fs";
import path from "node:path";
import { assert } from "vitest";
import { parseResult } from "./generated/schema.ts";

export function getTestResult() {
  const data = readFileSync(
    path.join(__dirname, "..", "..", "testdata", "result.json"),
  ).toString();

  const r = parseResult(data);
  assert.isNotNull(r);

  return r;
}
