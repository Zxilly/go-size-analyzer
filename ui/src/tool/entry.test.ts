import { readFileSync } from "node:fs";
import path from "node:path";
import { assert, describe, expect, expectTypeOf, it } from "vitest";
import { parseResult } from "../generated/schema.ts";
import type { EntryChildren, EntryLike, EntryType } from "./entry.ts";
import { BaseImpl, DisasmImpl, createEntry } from "./entry.ts";

describe("entry", () => {
  it("type should met children type", () => {
    expectTypeOf<EntryType>().toEqualTypeOf<keyof EntryChildren>();
  });

  it("match", () => {
    const data = readFileSync(
      path.join(__dirname, "..", "..", "..", "testdata", "result.json"),
    ).toString();

    const r = parseResult(data);
    assert.isNotNull(r);

    const e = createEntry(r);

    const matchEntry = <T extends EntryType>(e: EntryLike<T>) => {
      expect(e.getName()).toMatchSnapshot();
      expect(e.getType()).toMatchSnapshot();
      expect(e.getSize()).toMatchSnapshot();
      expect(e.getChildren().map((e => e.getName()))).toMatchSnapshot();
      expect(e.toString()).toMatchSnapshot();

      e.getChildren().forEach(e => matchEntry(e));
    };

    matchEntry(e);
  });

  it("baseImpl", () => {
    const i = new BaseImpl();

    expect(i.getID()).toBeTypeOf("number");
    expect(() => i.getName()).toThrow();
    expect(() => i.getURLSafeName()).toThrow();
  });

  describe("disasmImp", () => {
    it("getName returns expected name", () => {
      const disasm = new DisasmImpl("TestDisasm", 1024);
      expect(disasm.getName()).toBe("TestDisasm");
    });

    it("getSize returns correct size", () => {
      const disasm = new DisasmImpl("TestDisasm", 2048);
      expect(disasm.getSize()).toBe(2048);
    });

    it("getChildren returns empty array", () => {
      const disasm = new DisasmImpl("TestDisasm", 1024);
      expect(disasm.getChildren()).toEqual([]);
    });

    it("toString includes name and size", () => {
      const disasm = new DisasmImpl("TestDisasm", 1024);
      const str = disasm.toString();
      expect(str).toMatchSnapshot();
    });

    it("toString warns about potential size inaccuracy", () => {
      const disasm = new DisasmImpl("TestDisasm", 1024);
      const str = disasm.toString();
      expect(str).toContain("This size was not accurate.");
      expect(str).toContain("The real size determined by disassembling can be larger.");
    });

    it("getType returns 'disasm'", () => {
      const disasm = new DisasmImpl("TestDisasm", 1024);
      expect(disasm.getType()).toBe("disasm");
    });
  });
});
