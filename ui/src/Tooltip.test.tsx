import { render } from "@testing-library/react";
import { expect, it } from "vitest";
import { Tooltip } from "./Tooltip.tsx";
import type { Entry, EntryChildren, EntryType } from "./tool/entry.ts";

function getTestNode(): Entry {
  return {
    getChildren(): EntryChildren[EntryType] {
      return [];
    },
    getID(): number {
      return 1;
    },
    getName(): string {
      return "test";
    },
    getSize(): number {
      return 12345;
    },
    toString(): string {
      return "test content";
    },
    getType(): EntryType {
      return "unknown";
    },
    getURLSafeName(): string {
      return "test";
    },
  };
}

it("tooltip should render correctly when visible", () => {
  const { getByText } = render(
    <Tooltip
      visible
      node={getTestNode()}
    />,
  );
  expect(getByText("test")).toBeInTheDocument();
  expect(getByText("test content")).toBeInTheDocument();
});

it("tooltip should not render when not visible", () => {
  const r = render(
    <Tooltip
      visible={false}
      node={getTestNode()}
    />,
  );
    // should have a tooltip-hidden class
  expect(r.container.querySelector(".tooltip-hidden")).not.toBeNull();
});
