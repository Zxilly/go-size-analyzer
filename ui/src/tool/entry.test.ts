import {expectTypeOf, test} from "vitest";
import {EntryChildren, EntryType} from "./entry.ts";

test('entry type should met children type', () => {
    expectTypeOf<EntryType>().toEqualTypeOf<keyof EntryChildren>()
})