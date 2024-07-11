import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
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

describe("tooltip", () => {
  it("should render", () => {
    const { getByText } = render(
      <Tooltip
        x={0}
        y={0}
        node={getTestNode()}
      />,
    );
    expect(getByText("test")).toBeInTheDocument();
    expect(getByText("test content")).toBeInTheDocument();
  });

  it("should respond to position", () => {
    const r = render(
      <Tooltip
        x={0}
        y={0}
        node={getTestNode()}
      />,
    );
    const tooltip = r.container.querySelector<HTMLElement>(".tooltip");
    expect(tooltip).not.toBeNull();
    expect(tooltip!.style.left).toBe("10px");
    expect(tooltip!.style.top).toBe("30px");
  });
});
