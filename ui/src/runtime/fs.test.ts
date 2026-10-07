import { afterEach, describe, expect, it, vi } from "vitest";
import { flushLog, resetCallback, resetLog, setCallback } from "./fs";
import { MAX_LOG_LENGTH } from "./logLimit";

afterEach(resetCallback);

describe("WASM stdout bridge", () => {
  it("bounds a diagnostic burst, emits at most five messages, and preserves the final error", () => {
    const callback = vi.fn();
    setCallback(callback);
    const bytes = new TextEncoder().encode("malformed DWARF entry\n".repeat(100));
    for (let index = 0; index < 1000; index++) {
      expect(globalThis.fs.writeSync(2, bytes)).toBe(bytes.length);
    }
    globalThis.fs.writeSync(2, new TextEncoder().encode("Error: invalid binary\n"));
    expect(callback).toHaveBeenCalledTimes(4);
    flushLog();
    expect(callback).toHaveBeenCalledTimes(5);
    const log: string = callback.mock.calls[callback.mock.calls.length - 1][0];
    expect(log.length).toBeLessThanOrEqual(MAX_LOG_LENGTH);
    expect(log).toContain("[Earlier log output truncated]");
    expect(log).toMatch(/Error: invalid binary\n$/);
    flushLog();
    expect(callback).toHaveBeenCalledTimes(5);
    resetLog();
    globalThis.fs.writeSync(2, new TextEncoder().encode("new analysis\n"));
    flushLog();
    expect(callback).toHaveBeenLastCalledWith("new analysis\n");
  });

  it("bounds a single enormous write without requiring a newline", () => {
    const callback = vi.fn();
    setCallback(callback);
    const bytes = new TextEncoder().encode("x".repeat(MAX_LOG_LENGTH * 100));
    globalThis.fs.writeSync(2, bytes);
    flushLog();
    expect(callback.mock.calls[callback.mock.calls.length - 1][0].length).toBeLessThanOrEqual(MAX_LOG_LENGTH);
  });
});
