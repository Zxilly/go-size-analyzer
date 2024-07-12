import { readFileSync } from "node:fs";
import path from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { parseResult } from "../generated/schema.ts";
import { Explorer } from "./Explorer";

const result = parseResult(
  readFileSync(
    path.join(__dirname, "..", "..", "..", "testdata", "result.json"),
  ).toString(),
);

vi.mock("../../gsa.wasm?init", () => {
  return {
    default: async () => {
      return Promise.resolve({});
    },
  };
});

describe("explorer", () => {
  beforeEach(() => {
    vi.stubGlobal("Go", class {
      importObject = {};
      run = vi.fn(() => Promise.resolve());

      constructor() {
      }
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
    cleanup();
  });

  describe("wasm success", () => {
    beforeEach(() => {
      vi.stubGlobal("gsa_analyze", () => {
        return result;
      });
    });

    it("should display loading state initially", () => {
      render(<Explorer />);
      expect(screen.getByText("Loading WebAssembly module...")).toBeInTheDocument();
    });

    it("should display file selector when no file is selected", async () => {
      render(<Explorer />);
      await waitFor(() => screen.getByText("Select a go binary"));
    });

    it("should display analyzing state when a file is selected", async () => {
      render(<Explorer />);

      await waitFor(() => screen.getByText("Select a go binary"));

      fireEvent.change(screen.getByTestId("file-selector"), { target: { files: [new File(["it"], "test.bin")] } });
      await waitFor(() => screen.getByText("Analyzing test.bin"));
    });

    it("should display error when analysis fails", async () => {
      vi.stubGlobal("gsa_analyze", () => {
        return null;
      });

      render(<Explorer />);

      await waitFor(() => screen.getByText("Select a go binary"));

      fireEvent.change(screen.getByTestId("file-selector"), { target: { files: [new File(["test"], "test.bin")] } });
      await waitFor(() => screen.getByText("Failed to analyze test.bin"));
    });
  });
});
