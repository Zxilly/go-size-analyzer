import { it } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { Explorer } from "./Explorer.tsx";

it("explorer should display error when loading fails", async () => {
  render(<Explorer />);
  await waitFor(() => screen.getByText("Failed to load WebAssembly module"));
});
