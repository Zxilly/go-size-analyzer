import type { RefObject } from "react";
import { useCallback, useState } from "react";
import { useLifecycles } from "react-use";

export interface useMouseResult {
  clientX: number | null;
  clientY: number | null;
  isOver: boolean;
  eventTarget: EventTarget | null;
}

export function useMouse(ref: RefObject<SVGElement>): useMouseResult {
  const [clientX, setClientX] = useState<number | null>(null);
  const [clientY, setClientY] = useState<number | null>(null);
  const [eventTarget, setEventTarget] = useState<EventTarget | null>(null);

  const [isOver, setIsOver] = useState(false);

  const onMouseMove = useCallback((e: MouseEvent) => {
    setClientX(e.clientX);
    setClientY(e.clientY);

    if (e.target !== eventTarget) {
      setEventTarget(e.target);
    }
  }, [eventTarget]);

  const onMouseOver = useCallback(() => {
    setIsOver(true);
  }, []);

  const onMouseOut = useCallback(() => {
    setIsOver(false);
  }, []);

  const mount = useCallback(() => {
    if (ref.current) {
      ref.current.addEventListener("mousemove", onMouseMove);
      ref.current.addEventListener("mouseover", onMouseOver);
      ref.current.addEventListener("mouseout", onMouseOut);
    }
  }, [onMouseMove, onMouseOut, onMouseOver, ref]);

  const unmount = useCallback(() => {
    if (ref.current) {
      ref.current.removeEventListener("mousemove", onMouseMove);
      ref.current.removeEventListener("mouseover", onMouseOver);
      ref.current.removeEventListener("mouseout", onMouseOut);
    }
  }, [onMouseMove, onMouseOut, onMouseOver, ref]);

  useLifecycles(mount, unmount);

  return {
    clientX,
    clientY,
    isOver,
    eventTarget,
  };
}
