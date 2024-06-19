import {readFileSync} from "node:fs";
import path from "node:path";
import {assert, expect, expectTypeOf, test} from "vitest";
import {parseResult} from "../generated/schema.ts";
import {EntryChildren, EntryLike, EntryType, createEntry} from "./entry.ts";

test('entry type should met children type', () => {
    expectTypeOf<EntryType>().toEqualTypeOf<keyof EntryChildren>()
})

test('entry match', () => {
    const data = readFileSync(path.join(__dirname, '..', 'testdata', 'testdata.json')).toString();
    
    const r = parseResult(data)
    assert.isNotNull(r)
    
    const e = createEntry(r)
    
    const matchEntry = <T extends EntryType>(e: EntryLike<T>) => {
        expect(e.getName()).toMatchSnapshot()
        expect(e.getSize()).toMatchSnapshot()
        expect(e.toString()).toMatchSnapshot()

        e.getChildren().forEach(e => matchEntry(e))
    }
    
    matchEntry(e)
})