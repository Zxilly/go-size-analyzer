import type { Result } from "../schema/schema.ts";
import type { LoadEvent, WasmEvent } from "./event.ts";
import worker from "./worker.ts?worker&url";

export class GsaInstance {
  logHandler: (line: string) => void;
  worker: Worker;

  private constructor(worker: Worker, log: (line: string) => void) {
    this.worker = worker;
    this.logHandler = log;
  }

  static async create(log: (line: string) => void): Promise<GsaInstance> {
    const ret = new GsaInstance(
      new Worker(worker, {
        type: "module",
      }),
      log,
    );

    return new Promise((resolve, reject) => {
      const loadCb = (e: MessageEvent<LoadEvent>) => {
        const data = e.data;

        if (data.type !== "load") {
          return reject(new Error("Unexpected message type"));
        }

        ret.worker.removeEventListener("message", loadCb);
        if (data.status === "success") {
          resolve(ret);
        }
        else {
          reject(new Error(data.reason));
        }
      };

      ret.worker.addEventListener("message", loadCb);
    });
  }

  async analyze(filename: string, data: Uint8Array): Promise<Result | null> {
    return new Promise((resolve) => {
      const analyzeCb = (e: MessageEvent<WasmEvent>) => {
        const data = e.data;

        switch (data.type) {
          case "log":
            this.logHandler(data.line);
            break;
          case "analyze":
            this.worker.removeEventListener("message", analyzeCb);
            resolve(data.result);
        }
      };

      this.worker.addEventListener("message", analyzeCb);
      this.worker.postMessage([filename, data], [data.buffer]);
    });
  }
}
