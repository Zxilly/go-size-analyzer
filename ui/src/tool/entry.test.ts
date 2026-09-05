import type { EntryChildren, EntryLike, EntryType } from "./entry.ts";
import { describe, expect, expectTypeOf, it } from "vitest";
import { getTestResult } from "../test/testhelper.ts";
import { BaseImpl, createEntry, DisasmImpl, UnknownImpl } from "./entry.ts";

describe("entry", () => {
  it("type should met children type", () => {
    expectTypeOf<EntryType>().toEqualTypeOf<keyof EntryChildren>();
  });

  it("match", () => {
    const r = getTestResult();

    const e = createEntry(r);

    const matchEntry = <T extends EntryType>(e: EntryLike<T>) => {
      expect(e.getName()).toMatchSnapshot();
      expect(e.getType()).toMatchSnapshot();
      expect(e.getSize()).toMatchSnapshot();
      expect(e.getChildren().map(e => e.getName())).toMatchSnapshot();
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

  it("keeps debug and linker metadata visible in older reports", () => {
    const source = getTestResult();
    const section = source.sections[0];
    const entry = createEntry({
      ...source,
      size: 1200,
      packages: {},
      sections: [
        { ...section, name: "__debug_info", debug: true, file_size: 1000, known_size: 1000 },
        { ...section, name: ".symtab", debug: false, file_size: 200, known_size: 200 },
      ],
    });
    const children = entry.getChildren();
    expect(children.find(c => c.getURLSafeName() === "debug-sections")?.getSize()).toBe(1000);
    expect(children.find(c => c.getURLSafeName() === "metadata-sections")?.getSize()).toBe(200);
    expect(children.reduce((size, c) => size + c.getSize(), 0)).toBe(1200);
    expect(children.some(c => c.getType() === "unknown")).toBe(false);
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

  describe("unknownImpl", () => {
    it("getName returns 'Unknown'", () => {
      const unknown = new UnknownImpl(1024);
      expect(unknown.getName()).toBe("Unknown");
    });

    it("getSize returns correct size", () => {
      const unknown = new UnknownImpl(2048);
      expect(unknown.getSize()).toBe(2048);
    });

    it("getChildren returns empty array", () => {
      const unknown = new UnknownImpl(1024);
      expect(unknown.getChildren()).toEqual([]);
    });

    it("toString includes size and unknown part description", () => {
      const unknown = new UnknownImpl(1024);
      const str = unknown.toString();
      expect(str).toMatchSnapshot();
    });

    it("getType returns 'unknown'", () => {
      const unknown = new UnknownImpl(1024);
      expect(unknown.getType()).toBe("unknown");
    });
  });
});
