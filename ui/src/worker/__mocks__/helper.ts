import type { Result } from "../../schema/schema.ts";
import { getTestResult } from "../../test/testhelper.ts";

export class GsaInstance {
  log: any;

  private constructor(_worker: any, log: any) {
    this.log = log;
  }

  static async create(_log: (line: string) => void): Promise<GsaInstance> {
    return new GsaInstance({}, {});
  }

  async analyze(filename: string, _data: Uint8Array): Promise<Result | null> {
    if (filename === "fail") {
      return null;
    }

    for (let i = 0; i < 10; i++) {
      this.log(`Processing ${i}`);
    }

    return getTestResult();
  }
}
