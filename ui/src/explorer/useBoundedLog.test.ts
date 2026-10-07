import { act, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { MAX_LOG_LENGTH } from "../runtime/logLimit";
import { useBoundedLog } from "./useBoundedLog";

afterEach(() => vi.useRealTimers());

it("batches a worker burst into one render and bounds pending and displayed text", () => {
  vi.useFakeTimers();
  let renders = 0;
  const { result, unmount } = renderHook(() => {
    renders++;
    return useBoundedLog();
  });
  act(() => {
    for (let index = 0; index < 10000; index++) {
      result.current.appendLog("x".repeat(1000));
    }
    result.current.appendLog("Error: invalid binary");
  });
  expect(renders).toBe(1);
  expect(vi.getTimerCount()).toBe(1);
  act(() => vi.advanceTimersByTime(50));
  expect(renders).toBe(2);
  expect(result.current.log.length).toBe(MAX_LOG_LENGTH);
  expect(result.current.log).toMatch(/Error: invalid binary\n$/);
  act(() => result.current.appendLog("later"));
  unmount();
  expect(vi.getTimerCount()).toBe(0);
});
