import { render, screen, waitFor } from "@testing-library/react";
import { it } from "vitest";
import { Explorer } from "./Explorer.tsx";

it("explorer should display error when loading fails", async () => {
  render(<Explorer />);
  await waitFor(() => screen.getByText("Failed to load WebAssembly module"));
});
