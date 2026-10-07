import type { AnalyzeEvent, LoadEvent, LogEvent } from "./event.ts";
import gsa from "../../gsa.wasm?init";
import { flushLog, resetLog, setCallback } from "../runtime/fs";
import "../runtime/wasm_exec.js";

declare const self: DedicatedWorkerGlobalScope;
declare function gsa_analyze(name: string, data: Uint8Array): import("../schema/schema.ts").Result | null;

async function init() {
  const go = new Go();

  const inst = await gsa(go.importObject);

  go.run(inst).then(() => {
    console.error("Go exited");
  });
}

init().then(() => {
  self.postMessage({
    status: "success",
    type: "load",
  } satisfies LoadEvent);

  setCallback((line) => {
    self.postMessage({
      type: "log",
      line,
    } satisfies LogEvent);
  });
}).catch((e: Error) => {
  self.postMessage({
    status: "error",
    type: "load",
    reason: e.message,
  } satisfies LoadEvent);
});

self.onmessage = (e: MessageEvent<[string, Uint8Array]>) => {
  const [filename, data] = e.data;

  resetLog();
  let result: import("../schema/schema.ts").Result | null = null;
  try {
    result = gsa_analyze(filename, data);
  }
  catch (error) {
    const reason = error instanceof Error ? error.message : String(error);
    globalThis.fs.writeSync(2, new TextEncoder().encode(`Error: ${reason.slice(0, 4096)}\n`));
  }
  finally {
    flushLog();
  }

  self.postMessage({
    result,
    type: "analyze",
  } satisfies AnalyzeEvent);
};
