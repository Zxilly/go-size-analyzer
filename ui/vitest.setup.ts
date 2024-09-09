import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";
import "@testing-library/jest-dom/vitest";

afterEach(() => {
  cleanup();

  if (typeof window !== "undefined") {
    // cleanup jsdom
    window.location.hash = "";
    document.body.innerHTML = "";
    document.head.innerHTML = "";
  }
});
