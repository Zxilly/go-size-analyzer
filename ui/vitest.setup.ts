import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";
import "@testing-library/jest-dom/vitest";

vi.mock("@chenglou/pretext", () => ({
  prepareWithSegments: vi.fn((text: string) => ({ text })),
  layoutWithLines: vi.fn((_prepared: { text: string }, _maxWidth: number, lineHeight: number) => ({
    lineCount: 1,
    height: lineHeight,
    lines: [{ text: _prepared.text, width: _prepared.text.length * 7 }],
  })),
}));

afterEach(() => {
  cleanup();

  if (typeof window !== "undefined") {
    // cleanup jsdom
    window.location.hash = "";
    document.body.innerHTML = "";
    document.head.innerHTML = "";
  }
});
