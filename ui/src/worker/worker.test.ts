import { afterEach, expect, it, vi } from "vitest";
import { MAX_LOG_LENGTH } from "../runtime/logLimit";

vi.mock("../../gsa.wasm?init", () => ({ default: async () => ({}) }));
vi.mock("../runtime/wasm_exec.js", () => ({}));
afterEach(() => vi.unstubAllGlobals());

it("the actual worker bounds log messages, flushes failures, and resets each analysis", async () => {
  const postMessage = vi.fn();
  const worker = { postMessage, onmessage: null as null | ((event: MessageEvent) => void) };
  vi.stubGlobal("self", worker);
  vi.stubGlobal("Go", class {
    importObject = {};
    run() { return new Promise(() => {}); }
  });
  const analyze = vi.fn(() => {
    const bytes = new TextEncoder().encode("malformed DWARF entry\n".repeat(100));
    for (let index = 0; index < 1000; index++) {
      globalThis.fs.writeSync(2, bytes);
    }
    throw new Error("invalid binary");
  });
  vi.stubGlobal("gsa_analyze", analyze);
  await import("./worker");
  await vi.waitFor(() => expect(postMessage).toHaveBeenCalledWith({ type: "load", status: "success" }));
  postMessage.mockClear();
  worker.onmessage!(new MessageEvent("message", { data: ["bad", new Uint8Array()] }));
  const messages = postMessage.mock.calls.map(call => call[0]);
  const logs = messages.filter(message => message.type === "log");
  expect(logs).toHaveLength(5);
  expect(logs.every(log => log.line.length <= MAX_LOG_LENGTH)).toBe(true);
  expect(logs[logs.length - 1].line).toContain("Error: invalid binary");
  expect(messages[messages.length - 1]).toEqual({ type: "analyze", result: null });
  analyze.mockImplementation(() => {
    globalThis.fs.writeSync(2, new TextEncoder().encode("new analysis\n"));
    return null as never;
  });
  postMessage.mockClear();
  worker.onmessage!(new MessageEvent("message", { data: ["next", new Uint8Array()] }));
  expect(postMessage.mock.calls.map(call => call[0])).toEqual([
    { type: "log", line: "new analysis\n" },
    { type: "analyze", result: null },
  ]);
});
