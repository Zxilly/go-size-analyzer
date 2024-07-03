import { readFileSync } from "node:fs";
import path from "node:path";
import { assert, expect, expectTypeOf, it } from "vitest";
import { parseResult } from "../generated/schema.ts";
import type { EntryChildren, EntryLike, EntryType } from "./entry.ts";
import { createEntry } from "./entry.ts";

it("entry type should met children type", () => {
  expectTypeOf<EntryType>().toEqualTypeOf<keyof EntryChildren>();
});

it("entry match", () => {
  const data = readFileSync(
    path.join(__dirname, "..", "..", "..", "testdata", "result.json"),
  ).toString();

  const r = parseResult(data);
  assert.isNotNull(r);

  const e = createEntry(r);

  const matchEntry = <T extends EntryType>(e: EntryLike<T>) => {
    expect(e.getName()).toMatchSnapshot();
    expect(e.getSize()).toMatchSnapshot();
    expect(e.toString()).toMatchSnapshot();

    e.getChildren().forEach(e => matchEntry(e));
  };

  matchEntry(e);
});
