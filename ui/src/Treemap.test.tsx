import {readFileSync} from "fs";
import path from "node:path";
import {expect, test} from "vitest";
import {render} from '@testing-library/react'
import {parseResult} from "./generated/schema.ts";
import {createEntry} from "./tool/entry.ts";
import TreeMap from "./TreeMap.tsx";

test("Treemap", () => {
    const data = readFileSync(path.join(__dirname, '..', '..', 'testdata', 'result.json')).toString();

    const r = parseResult(data)
    expect(r).toBeDefined()

    const e = createEntry(r)
    expect(e).toMatchSnapshot()

    const rr = render(
        <TreeMap entry={e}/>
    )

    expect(rr.container).toMatchSnapshot()
})