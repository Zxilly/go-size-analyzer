// @vitest-environment node

import "@vitest/web-worker";

import { readFile } from "node:fs/promises";
import path from "node:path";
import { describe, expect, it, vi } from "vitest";
import { GsaInstance } from "./helper.ts";
import "../runtime/wasm_exec.js";

vi.mock("../../gsa.wasm?init", async () => {
  const buffer = await readFile(path.join(__dirname, "../../gsa.wasm"));
  // create blob
  return {
    default: async (i: WebAssembly.Imports) => {
      return (await WebAssembly.instantiate(buffer, i)).instance;
    },
  };
});

describe("worker helper", () => {
  it("instance", async () => {
    const logHandler = vi.fn((l: string) => {
      console.log(l);
    });

    const inst = await GsaInstance.create(logHandler);

    const data = new Uint8Array([1, 2, 3, 4]);

    const result = await inst.analyze("test.bin", data);

    expect(logHandler).toHaveBeenCalled();
    expect(result).toBeNull();
  });
});
