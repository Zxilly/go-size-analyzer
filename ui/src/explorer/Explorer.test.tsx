import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Explorer } from "./Explorer";

vi.mock("../worker/helper.ts");

describe("explorer", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    cleanup();
  });

  describe("wasm success", () => {
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
      render(<Explorer />);

      await waitFor(() => screen.getByText("Select a go binary"));

      fireEvent.change(screen.getByTestId("file-selector"), { target: { files: [new File(["test"], "fail")] } });
      await waitFor(() => screen.getByText("Failed to analyze fail"));
    });
  });
});
