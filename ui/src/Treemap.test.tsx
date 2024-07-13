import { expect, it } from "vitest";
import { render } from "@testing-library/react";
import { createEntry } from "./tool/entry.ts";
import TreeMap from "./TreeMap.tsx";
import { getTestResult } from "./testhelper.ts";

it("treemap", () => {
  const r = getTestResult();

  const e = createEntry(r);
  expect(e).toMatchSnapshot();

  const rr = render(
    <TreeMap entry={e} />,
  );

  expect(rr.container).toMatchSnapshot();
});
