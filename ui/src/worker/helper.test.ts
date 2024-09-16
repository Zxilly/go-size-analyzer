// @vitest-environment node

import { readFile } from "node:fs/promises";
import path from "node:path";
import { describe, expect, it, vi } from "vitest";
import createFetchMock from "vitest-fetch-mock";
import { GsaInstance } from "./helper.ts";
import "@vitest/web-worker";
import "../runtime/wasm_exec.js";

const fetchMocker = createFetchMock(vi);
fetchMocker.enableMocks();

// @ts-expect-error the mocker got the wrong type
fetchMocker.mockResponse(async () => {
  const buffer = await readFile(path.join(__dirname, "../../gsa.wasm"));
  return new Response(buffer, {
    status: 200,
    headers: { "Content-Type": "application/octet-stream" },
  });
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
