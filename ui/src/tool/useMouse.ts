import type { RefObject } from "react";
import { useCallback, useState } from "react";
import { useLifecycles } from "react-use";
import type Konva from "konva";
import type { KonvaEventListener, KonvaEventObject } from "konva/lib/Node";
import type { Stage } from "konva/lib/Stage";

export interface useMouseResult {
  clientX: number | null;
  clientY: number | null;
  isOver: boolean;
  eventTarget: KonvaEventObject<MouseEvent> | null;
}

export function useMouse(ref: RefObject<Konva.Stage>): useMouseResult {
  const [clientX, setClientX] = useState<number | null>(null);
  const [clientY, setClientY] = useState<number | null>(null);
  const [eventObject, setEventObject] = useState<KonvaEventObject<MouseEvent> | null>(null);

  const [isOver, setIsOver] = useState(false);

  const onMouseMove: KonvaEventListener<Stage, MouseEvent> = useCallback((e) => {
    if (!ref.current) {
      return;
    }

    setClientX(e.evt.clientX);
    setClientY(e.evt.clientY);

    if (eventObject?.currentTarget._id !== e.currentTarget._id) {
      console.log("new event object");
      setEventObject(e);
    }
  }, [eventObject, ref]);

  const onMouseOver = useCallback(() => {
    setIsOver(true);
  }, []);

  const onMouseOut = useCallback(() => {
    setIsOver(false);
  }, []);

  const mount = useCallback(() => {
    if (ref && ref.current) {
      ref.current.on("mousemove", onMouseMove);
      ref.current.on("mouseover", onMouseOver);
      ref.current.on("mouseout", onMouseOut);
    }
  }, [onMouseMove, onMouseOut, onMouseOver, ref]);

  const unmount = useCallback(() => {
    if (ref && ref.current) {
      ref.current.off("mousemove", onMouseMove);
      ref.current.off("mouseover", onMouseOver);
      ref.current.off("mouseout", onMouseOut);
    }
  }, [onMouseMove, onMouseOut, onMouseOver, ref]);

  useLifecycles(mount, unmount);

  return {
    clientX,
    clientY,
    isOver,
    eventTarget: eventObject,
  };
}
