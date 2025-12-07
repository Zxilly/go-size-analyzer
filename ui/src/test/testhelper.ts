import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { assert } from "vitest";
import { parseResult } from "../schema/schema.ts";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export function getTestResult() {
  const data = readFileSync(
    join(__dirname, "..", "..", "..", "testdata", "result.json"),
  ).toString();

  const r = parseResult(data);
  assert.isNotNull(r);

  return r!;
}
