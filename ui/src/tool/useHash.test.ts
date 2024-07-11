import { act, renderHook } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";
import { useHash } from "./useHash";

describe("useHash", () => {
  beforeAll(() => {
    window.location = { hash: "" } as any;
  });

  it("initializes with the current window location hash", () => {
    window.location.hash = "#initial";
    const { result } = renderHook(() => useHash());
    expect(result.current[0]).toBe("#initial");
  });

  it("updates hash when setHash is called with a new value", () => {
    const { result } = renderHook(() => useHash());
    act(() => {
      result.current[1]("#newHash");
    });
    expect(window.location.hash).toBe("#newHash");
  });

  it("does not update hash if setHash is called with the current hash value", () => {
    window.location.hash = "#sameHash";
    const { result } = renderHook(() => useHash());
    act(() => {
      result.current[1]("#sameHash");
    });
    expect(window.location.hash).toBe("#sameHash");
  });

  it("removes hash when setHash is called with an empty string", () => {
    window.location.hash = "#toBeRemoved";
    const { result } = renderHook(() => useHash());
    act(() => {
      result.current[1]("");
    });
    expect(window.location.hash).toBe("");
  });

  it("responds to window hashchange event", () => {
    const { result } = renderHook(() => useHash());
    act(() => {
      window.location.hash = "#changedViaEvent";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });
    expect(result.current[0]).toBe("#changedViaEvent");
  });
});
