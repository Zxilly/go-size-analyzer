import type { Result } from "../schema/schema.ts";
import { parseResult } from "../schema/schema.ts";

export function loadDataFromEmbed(): Result {
  const doc = document.querySelector("#data")!;
  if (doc === null) {
    throw new Error("Failed to find data element");
  }

  const ret = parseResult(doc.textContent!);
  if (ret === null) {
    throw new Error("Failed to parse data");
  }
  return ret;
}

export function formatBytes(bytes: number) {
  if (bytes === 0)
    return "0 B";
  const k = 1024;
  const dm = 2;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${Number.parseFloat((bytes / k ** i).toFixed(dm))} ${sizes[i]}`;
}

export function title(s: string): string {
  return s[0].toUpperCase() + s.slice(1);
}

export function trimPrefix(str: string, prefix: string) {
  if (str.startsWith(prefix)) {
    return str.slice(prefix.length);
  }
  return str;
}
