import "../runtime/wasm_exec.js";
import { setCallback } from "../runtime/fs";
import gsa from "../../gsa.wasm?url";
import type { AnalyzeEvent, LoadEvent, LogEvent } from "./event.ts";

declare const self: DedicatedWorkerGlobalScope;
declare function gsa_analyze(name: string, data: Uint8Array): import("../generated/schema.ts").Result;

async function init() {
  const go = new Go();

  const inst = (await WebAssembly.instantiateStreaming(
    fetch(gsa),
    go.importObject,
  )).instance;

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
  const result = gsa_analyze(filename, data);

  self.postMessage({
    result,
    type: "analyze",
  } satisfies AnalyzeEvent);
};
