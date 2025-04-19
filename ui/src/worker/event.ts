export interface LoadEvent {
  type: "load";
  status: "success" | "error";
  reason?: string;
}

export interface AnalyzeEvent {
  type: "analyze";
  result: import("../schema/schema.ts").Result | null;
}

export interface LogEvent {
  type: "log";
  line: string;
}

export type WasmEvent = LoadEvent | AnalyzeEvent | LogEvent;
