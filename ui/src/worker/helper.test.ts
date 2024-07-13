import "@vitest/web-worker";
import { describe, expect, it, vi } from "vitest";
import { GsaInstance } from "./helper.ts";

describe("worker helper", () => {
  it.skip("instance", async () => {
    const logHandler = vi.fn();

    const inst = await GsaInstance.create(logHandler);

    const data = new Uint8Array([1, 2, 3, 4]);

    const result = await inst.analyze("test.bin", data);

    expect(logHandler).toHaveBeenCalled();
    expect(result).toBeNull();
  });
});
