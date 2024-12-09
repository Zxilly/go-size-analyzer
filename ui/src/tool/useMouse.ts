import type { RefObject } from "react";
import { useEffect, useState } from "react";

export interface useMouseResult {
  clientX: number | null;
  clientY: number | null;
  isOver: boolean;
  eventTarget: EventTarget | null;
}

export function useMouse(ref: RefObject<SVGElement | null>): useMouseResult {
  const [clientX, setClientX] = useState<number | null>(null);
  const [clientY, setClientY] = useState<number | null>(null);
  const [eventTarget, setEventTarget] = useState<EventTarget | null>(null);

  const [isOver, setIsOver] = useState(false);

  useEffect(() => {
    const abort = new AbortController();
    const signal = abort.signal;

    if (ref.current) {
      ref.current.addEventListener("mousemove", (e) => {
        setClientX(e.clientX);
        setClientY(e.clientY);

        if (e.target !== eventTarget) {
          setEventTarget(e.target);
        }
      }, { signal });
      ref.current.addEventListener("mouseover", () => {
        setIsOver(true);
      }, { signal });
      ref.current.addEventListener("mouseout", () => {
        setIsOver(false);
      }, { signal });
    }

    return () => {
      abort.abort();
    };
  }, [eventTarget, ref]);

  return {
    clientX,
    clientY,
    isOver,
    eventTarget,
  };
}
