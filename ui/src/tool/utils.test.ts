import { describe, expect, it } from "vitest";
import { getTestResult } from "../test/testhelper.ts";
import { formatBytes, loadDataFromEmbed, title, trimPrefix } from "./utils.ts";

it("formatBytes should correctly format bytes into human readable format", () => {
  expect(formatBytes(0)).toBe("0 B");
  expect(formatBytes(1024)).toBe("1 KB");
  expect(formatBytes(1048576)).toBe("1 MB");
});

it("title should capitalize the first letter of the string", () => {
  expect(title("hello")).toBe("Hello");
  expect(title("world")).toBe("World");
});

it("trimPrefix should remove the prefix from the string", () => {
  expect(trimPrefix("HelloWorld", "Hello")).toBe("World");
  expect(trimPrefix("HelloWorld", "World")).toBe("HelloWorld");
});

describe("loadDataFromEmbed", () => {
  it("should return parsed data when data is correctly formatted", () => {
    const data = getTestResult();

    document.body.innerHTML = `<div id="data">${JSON.stringify(data)}</div>`;
    expect(() => loadDataFromEmbed()).not.toThrow();
  });

  it("should throw error when element with id data is not found", () => {
    document.body.innerHTML = ""; // No element with id="data"
    expect(() => loadDataFromEmbed()).toThrow("Failed to find data element");
  });

  it("should throw error when data is null", () => {
    document.body.innerHTML = `<div id="data">{}</div>`;
    expect(() => loadDataFromEmbed()).toThrow("Failed to parse data");
  });

  it("should throw error when data is not parsable", () => {
    document.body.innerHTML = `<div id="data">unparsable data</div>`;
    expect(() => loadDataFromEmbed()).toThrow(`Unexpected token 'u', "unparsable data" is not valid JSON`);
  });
});
