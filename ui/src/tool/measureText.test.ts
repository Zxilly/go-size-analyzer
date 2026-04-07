import { describe, expect, it } from "vitest";
import { getScale, getShortName, measureText } from "./measureText.ts";

describe("measureText", () => {
  it("returns width based on text length and fixed height", () => {
    const [width, height] = measureText("hello");
    expect(width).toBe(35); // 5 * 7
    expect(height).toBe(12.8);
  });

  it("returns zero width for empty string", () => {
    const [width, height] = measureText("");
    expect(width).toBe(0);
    expect(height).toBe(12.8);
  });
});

describe("getShortName", () => {
  it("returns last segment of a path", () => {
    expect(getShortName("runtime/symtab.go")).toBe("symtab.go");
  });

  it("handles backslash separators", () => {
    expect(getShortName("runtime\\symtab.go")).toBe("symtab.go");
  });

  it("returns parent/version for version-like last segment", () => {
    expect(getShortName("github.com/foo/bar/v2")).toBe("bar/v2");
  });

  it("returns the title itself if no separator", () => {
    expect(getShortName("main.go")).toBe("main.go");
  });

  it("handles deeply nested paths", () => {
    expect(getShortName("a/b/c/d/e.go")).toBe("e.go");
  });

  it("returns parent\\version with backslash", () => {
    expect(getShortName("foo\\bar\\v3")).toBe("bar\\v3");
  });
});

describe("getScale", () => {
  it("returns empty string and 0 for empty title", () => {
    const [display, scale] = getScale("", 100, 100, false);
    expect(display).toBe("");
    expect(scale).toBe(0);
  });

  it("returns title and valid scale for normal case", () => {
    // "hi" → width=14, height=12.8
    // scale = min(100*0.9/14, 100*0.9/12.8) = min(6.43, 7.03) = 6.43
    // scale > 1 → sqrt(6.43) ≈ 2.536
    const [display, scale] = getScale("hi", 100, 100, false);
    expect(display).toBe("hi");
    expect(scale).toBeGreaterThan(1);
  });

  it("caps scale at 1 for hasChildren", () => {
    const [display, scale] = getScale("hi", 100, 100, true);
    expect(display).toBe("hi");
    expect(scale).toBeLessThanOrEqual(1);
  });

  it("falls back to short name when scale < 0.7", () => {
    // "very/long/deeply/nested/path/name.go" → width = 37*7 = 259
    // with width=20: scale = 20*0.9/259 ≈ 0.069, < 0.7 → fallback to "name.go"
    const [display] = getScale("very/long/deeply/nested/path/name.go", 20, 100, false);
    expect(display).toBe("name.go");
  });

  it("applies sqrt when scale > 1 for non-children nodes", () => {
    // "ab" → width=14, height=12.8
    // width=200, height=200: scale = min(200*0.9/14, 200*0.9/12.8) = min(12.86, 14.06) = 12.86
    // sqrt(12.86) ≈ 3.586
    const [, scale] = getScale("ab", 200, 200, false);
    expect(scale).toBeCloseTo(Math.sqrt(Math.min((200 * 0.9) / 14, (200 * 0.9) / 12.8)), 5);
  });

  it("handles Infinity scale (zero-width text)", () => {
    // Empty text already handled, but test with a very wide container and short text
    const [, scale] = getScale("a", 10000, 10000, false);
    expect(Number.isFinite(scale)).toBe(true);
    expect(scale).toBeGreaterThan(0);
  });
});
