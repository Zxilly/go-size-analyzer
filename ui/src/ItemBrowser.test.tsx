import type { Result } from "./schema/schema.ts";
import { fireEvent, render, within } from "@testing-library/react";
import { expect, it } from "vitest";
import { getTestResult } from "./test/testhelper.ts";
import { createEntry } from "./tool/entry.ts";
import TreeMap from "./TreeMap.tsx";

function denseReport(): Result {
  const source = getTestResult();
  return {
    ...source,
    name: "dense",
    size: 100_060,
    coverage: undefined,
    sections: [
      {
        ...source.sections[0],
        name: ".text",
        debug: false,
        file_size: 100_060,
        known_size: 100_060,
      },
    ],
    packages: {
      example: {
        name: "example",
        type: "main",
        size: 100_060,
        symbols: [],
        subPackages: {},
        files: [
          {
            file_path: "source.go",
            functions: [
              {
                name: "Large",
                full_name: "example.Large",
                code_size: 100_000,
                addr: 1,
              },
              ...Array.from({ length: 60 }, (_, index) => ({
                name: `Tiny${index}`,
                full_name: `example.Tiny${index}`,
                code_size: 1,
                addr: index + 2,
              })),
            ],
          },
        ],
      },
    },
  };
}

it("opens culled functions through a bounded searchable list and restores their hash", () => {
  const report = denseReport();
  const entry = createEntry(report);
  const view = render(<TreeMap entry={entry} />);
  fireEvent.click(view.getByText("source.go"));
  expect(view.queryByText("example.Tiny59")).not.toBeInTheDocument();
  expect(view.getByText("example.Large")).toBeInTheDocument();
  fireEvent.click(view.getByRole("button", { name: "Browse all 61 items" }));
  const browser = within(view.getByRole("complementary", { name: "Browse items" }));
  expect(browser.getAllByRole("listitem")).toHaveLength(50);
  fireEvent.click(browser.getByRole("button", { name: "Next" }));
  expect(browser.getAllByRole("listitem")).toHaveLength(11);
  fireEvent.change(browser.getByRole("searchbox"), {
    target: { value: "Tiny59" },
  });
  expect(browser.getAllByRole("listitem")).toHaveLength(1);
  fireEvent.click(browser.getByRole("button", { name: "example.Tiny59 1 B" }));
  expect(view.getByRole("heading", { name: "example.Tiny59" })).toBeInTheDocument();
  const hash = window.location.hash;
  expect(hash).toBe("#dense#main-packages#example#source.go#example.Tiny59%403d");
  expect(entry.getSize()).toBe(100_060);
  view.unmount();

  const restored = render(<TreeMap entry={createEntry(report)} />);
  expect(restored.getByRole("heading", { name: "example.Tiny59" })).toBeInTheDocument();
  expect(window.location.hash).toBe(hash);
  fireEvent.click(restored.getByRole("button", { name: "Back to file" }));
  expect(restored.queryByRole("region", { name: "Function details" })).not.toBeInTheDocument();
  expect(restored.getByRole("button", { name: "Browse all 61 items" })).toBeInTheDocument();
  fireEvent.click(restored.getByText("example"));
  expect(window.location.hash).toBe("#dense#main-packages#example");
});

it("drops paths from a previous report when Explorer replaces its entry", () => {
  const report = denseReport();
  const view = render(<TreeMap entry={createEntry(report)} />);
  fireEvent.click(view.getByText("source.go"));
  fireEvent.click(view.getByText("example.Large"));
  expect(view.getByRole("heading", { name: "example.Large" })).toBeInTheDocument();
  view.rerender(<TreeMap entry={createEntry({ ...report, name: "replacement", packages: {} })} />);
  expect(view.queryByRole("region", { name: "Function details" })).not.toBeInTheDocument();
  expect(view.getByText("replacement")).toBeInTheDocument();
  expect(window.location.hash).toBe("");
});
