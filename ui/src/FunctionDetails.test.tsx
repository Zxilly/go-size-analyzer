import type { FileFunction, Result } from "./schema/schema.ts";
import { fireEvent, render, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FunctionDetails } from "./FunctionDetails.tsx";
import { getTestResult } from "./test/testhelper.ts";
import { createEntry, FunctionImpl } from "./tool/entry.ts";
import TreeMap from "./TreeMap.tsx";

const fn: FileFunction = {
  name: "Read",
  full_name: "example.Read",
  addr: 4096,
  code_size: 64,
  pcln_bytes: 16,
  referenced_bytes: 32,
  reference_evidence_count: 12,
  reference_evidence: [
    {
      offset: 512,
      size: 32,
      section: ".rodata",
      source: "static_read",
      role: "reference",
      boundary: "instruction",
    },
  ],
};

afterEach(() => {
  window.location.hash = "";
});

describe("function details", () => {
  it("shows bounded physical evidence with explicit shared-reference semantics", () => {
    const close = vi.fn();
    const view = render(<FunctionDetails entry={new FunctionImpl(fn, "source.go")} onClose={close} />);
    expect(view.getByRole("region", { name: "Function details" })).toBeInTheDocument();
    expect(view.getByRole("heading", { name: "example.Read" })).toBeInTheDocument();
    expect(view.getByText("0x200–0x220")).toBeInTheDocument();
    expect(view.getByText("static_read")).toBeInTheDocument();
    expect(view.getByText("reference · instruction")).toBeInTheDocument();
    expect(view.getByText(/Showing 1 of 12/)).toBeInTheDocument();
    expect(view.getByText(/does not measure potential savings/)).toBeInTheDocument();
    fireEvent.click(view.getByRole("button", { name: "Back to file" }));
    expect(close).toHaveBeenCalledOnce();
  });

  it.each([
    [undefined, "Reference analysis is unavailable in this report."],
    [0, "No referenced file bytes were identified."],
    [32, "This report has no evidence samples. Generate it with --coverage-details to include them."],
  ])("explains missing samples when referenced_bytes is %s", (bytes, message) => {
    const view = render(<FunctionDetails entry={new FunctionImpl({ ...fn, referenced_bytes: bytes as number | undefined, reference_evidence: undefined }, "source.go")} onClose={() => {}} />);
    expect(view.getByText(message as string)).toBeInTheDocument();
  });

  it("drills from a file into functions and opens persistent evidence", () => {
    const data: Result = {
      ...getTestResult(),
      name: "fixture",
      size: 80,
      coverage: undefined,
      sections: [
        {
          ...getTestResult().sections[0],
          name: ".text",
          debug: false,
          size: 80,
          file_size: 80,
          known_size: 80,
        },
      ],
      packages: {
        example: {
          name: "example",
          type: "main",
          size: 80,
          subPackages: {},
          symbols: [],
          files: [
            {
              file_path: "source.go",
              size: 80,
              pcln_size: 16,
              functions: [fn],
            },
          ],
        },
      },
    };
    const view = render(<TreeMap entry={createEntry(data)} />);
    expect(view.queryByText("example.Read")).not.toBeInTheDocument();
    fireEvent.click(view.getByText("source.go"));
    fireEvent.click(view.getByText("example.Read"));
    const details = view.getByRole("region", { name: "Function details" });
    expect(view.container.querySelector(".tooltip")).toBeNull();
    expect(within(details).getByText("static_read")).toBeInTheDocument();
    expect(window.location.hash).toContain("example.Read%401000");
    fireEvent.click(within(details).getByRole("button", { name: "Back to file" }));
    expect(view.queryByRole("region", { name: "Function details" })).not.toBeInTheDocument();
    expect(view.getByText("example.Read")).toBeInTheDocument();
  });
});
