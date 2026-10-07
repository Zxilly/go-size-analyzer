import { useCallback, useEffect, useRef, useState } from "react";
import { MAX_LOG_LENGTH } from "../runtime/logLimit";

/** Bound pending work as well as rendered logs, even if a worker sends a burst. */
export function useBoundedLog() {
  const [log, setLog] = useState("");
  const pendingRef = useRef("");
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const appendLog = useCallback((line: string) => {
    pendingRef.current = (`${pendingRef.current + line.slice(-MAX_LOG_LENGTH)}\n`).slice(-MAX_LOG_LENGTH);
    if (timerRef.current === undefined) {
      timerRef.current = setTimeout(() => {
        const batch = pendingRef.current;
        pendingRef.current = "";
        timerRef.current = undefined;
        setLog(previous => (previous + batch).slice(-MAX_LOG_LENGTH));
      }, 50);
    }
  }, []);

  useEffect(() => () => {
    clearTimeout(timerRef.current);
    timerRef.current = undefined;
    pendingRef.current = "";
  }, []);
  return { log, appendLog };
}
