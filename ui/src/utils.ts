import {Result} from "./schema.ts";

export function loadData(): Result {
    const doc = document.querySelector("#data")!;
    return JSON.parse(doc.textContent!) as Result;
}
