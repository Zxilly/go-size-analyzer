import { describe, expect, it } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import { createEntry } from "./tool/entry.ts";
import TreeMap from "./TreeMap.tsx";
import { getTestResult } from "./test/testhelper.ts";

function getTestEntry() {
  return createEntry(getTestResult());
}

describe("treeMap", () => {
  it("render snapshot", () => {
    const rr = render(
      <TreeMap entry={getTestEntry()} />,
    );

    expect(rr.container).toMatchSnapshot();
  });

  it("render with hash", () => {
    window.location.hash = "#bin-linux-1.21-amd64#std-packages#runtime#runtime1.go";

    const { getByText } = render(
      <TreeMap entry={getTestEntry()} />,
    );

    const rect = getByText("runtime1.go");
    expect(rect).toBeInTheDocument();

    const strokeEle = rect.parentElement?.children.item(0);
    expect(strokeEle).not.toBeNull();

    // <rect fill="rgb(80, 149, 149)" rx="2" ry="2" width="1780" height="1002" stroke="#fff" stroke-width="2"></rect>
    expect(strokeEle).toHaveAttribute("stroke", "#fff");
    expect(strokeEle).toHaveAttribute("stroke-width", "2");
  });

  it("auto set hash", () => {
    const { getByText } = render(
      <TreeMap entry={getTestEntry()} />,
    );

    const rect = getByText("symtab.go");

    fireEvent.click(rect);

    expect(window.location.hash).toBe("#bin-linux-1.21-amd64#std-packages#runtime#symtab.go");
  });

  it("accept invalid hash", () => {
    window.location.hash = "#invalid-hash";

    const { getByText } = render(
      <TreeMap entry={getTestEntry()} />,
    );

    expect(getByText("Main Packages Size")).not.toBeNull();
    expect(getByText("Std Packages Size")).not.toBeNull();
  });

  it("should handle move event", () => {
    const { getByText } = render(
      <TreeMap entry={getTestEntry()} />,
    );

    const rect = getByText("symtab.go");

    fireEvent.mouseMove(rect);

    const tooltip = document.querySelector(".tooltip");
    expect(tooltip).not.toBeNull();
  });
});
