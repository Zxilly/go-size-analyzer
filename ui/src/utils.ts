import {Result} from "./schema/schema.ts";
import {parseResult} from "./generated/schema.ts";

export function loadData(): Result {
    const doc = document.querySelector("#data")!;
    return parseResult(doc.textContent!);
}
