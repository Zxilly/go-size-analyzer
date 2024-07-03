import { fireEvent, render } from "@testing-library/react";
import { assert, expect, it } from "vitest";
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

it("tooltip should update position on mouse move", () => {
  const { getByText } = render(<Tooltip visible node={getTestNode()} />);
  fireEvent.mouseMove(document, { clientX: 100, clientY: 100 });
  const tooltip = getByText("test").parentElement;
  if (!tooltip) {
    assert.isNotNull(tooltip);
    return;
  }

  expect(tooltip.style.left).toBe("110px");
  expect(tooltip.style.top).toBe("130px");
});
