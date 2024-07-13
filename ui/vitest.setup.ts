import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

afterEach(() => {
  cleanup();

  // cleanup jsdom
  window.location.hash = "";
  document.body.innerHTML = "";
  document.head.innerHTML = "";
});
