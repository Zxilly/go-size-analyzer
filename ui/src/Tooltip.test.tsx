import type { Entry, EntryChildren, EntryType } from "./tool/entry.ts";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { describe, expect, it } from "vitest";
import { Tooltip } from "./Tooltip.tsx";

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

function getFakeRef(): React.RefObject<SVGTextElement> {
  const base = document.createElement("div");
  document.body.appendChild(base);

  return {
    current: base as any,
  };
}

describe("tooltip", () => {
  it("should render", async () => {
    const ref = getFakeRef();

    const { getByText } = render(
      <Tooltip
        moveRef={ref}
        getTargetNode={() => getTestNode()}
      />,
    );

    await userEvent.hover(ref.current!);
    fireEvent.mouseOver(ref.current!);
    fireEvent.mouseMove(ref.current!);

    expect(getByText("test")).toBeInTheDocument();
    expect(getByText("test content")).toBeInTheDocument();
  });

  it("should respond to position", async () => {
    const ref = getFakeRef();

    const r = render(
      <Tooltip
        moveRef={ref}
        getTargetNode={() => getTestNode()}
      />,
    );

    await userEvent.hover(ref.current!);
    fireEvent.mouseOver(ref.current!);
    fireEvent.mouseMove(ref.current!, { clientX: 10, clientY: 30 });

    const tooltip = r.container.querySelector<HTMLElement>(".tooltip");
    expect(tooltip).not.toBeNull();
    expect(tooltip!.style.left).toBe("20px");
    expect(tooltip!.style.top).toBe("60px");
  });

  it("should auto shift", async () => {
    const ref = getFakeRef();

    const r = render(
      <Tooltip
        moveRef={ref}
        getTargetNode={() => getTestNode()}
      />,
    );

    await userEvent.hover(ref.current!);
    fireEvent.mouseOver(ref.current!);
    fireEvent.mouseMove(ref.current!, { clientX: window.innerWidth - 1, clientY: window.innerHeight - 1 });

    const tooltip = r.container.querySelector<HTMLElement>(".tooltip");
    expect(tooltip).not.toBeNull();

    const boundingRect = tooltip!.getBoundingClientRect();
    expect(boundingRect.right).toBeLessThanOrEqual(window.innerWidth);
    expect(boundingRect.bottom).toBeLessThanOrEqual(window.innerHeight);
  });
});
