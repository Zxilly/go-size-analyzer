import type { Result } from "../../generated/schema.ts";
import { getTestResult } from "../../test/testhelper.ts";

export class GsaInstance {
  private constructor(_worker: any, _log: any) {
  }

  static async create(_log: (line: string) => void): Promise<GsaInstance> {
    return new GsaInstance({}, {});
  }

  async analyze(filename: string, _data: Uint8Array): Promise<Result | null> {
    if (filename === "fail") {
      return null;
    }

    await new Promise(resolve => setTimeout(resolve, 100));

    return getTestResult();
  }
}
