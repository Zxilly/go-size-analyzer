import type { RefObject } from "react";
import type { Entry } from "./tool/entry.ts";
import React, { useLayoutEffect, useMemo, useRef } from "react";
import { useMouse } from "./tool/useMouse.ts";

const Tooltip_marginX = 10;
const Tooltip_marginY = 30;

export interface TooltipProps {
  moveRef: RefObject<SVGElement | null>;
  getTargetNode: (e: EventTarget) => Entry | null;
}

export const Tooltip: React.FC<TooltipProps>
  = ({
    moveRef,
    getTargetNode,
  }) => {
    const ref = useRef<HTMLDivElement>(null);

    const {
      clientX: x,
      clientY: y,
      isOver,
      eventTarget: mouseEventTarget,
    } = useMouse(moveRef);

    const node = useMemo(() => {
      if (!mouseEventTarget) {
        return null;
      }

      return getTargetNode(mouseEventTarget);
    }, [getTargetNode, mouseEventTarget]);

    const path = useMemo(() => {
      return node?.getName() ?? "";
    }, [node]);

    const content = useMemo(() => {
      return node?.toString() ?? "";
    }, [node]);

    useLayoutEffect(() => {
      if (ref.current && x && y && isOver && node) {
        const pos = {
          left: x + Tooltip_marginX,
          top: y + Tooltip_marginY,
        };

        const boundingRect = ref.current.getBoundingClientRect();

        if (pos.left + boundingRect.width > window.innerWidth) {
          // Shifting horizontally
          pos.left = window.innerWidth - boundingRect.width;
        }

        if (pos.top + boundingRect.height > window.innerHeight) {
          // Flipping vertically
          pos.top = y - Tooltip_marginY - boundingRect.height;
        }

        ref.current.style.left = `${pos.left}px`;
        ref.current.style.top = `${pos.top}px`;
        ref.current.style.visibility = "visible";
      }
      else if (ref.current) {
        ref.current.style.visibility = "hidden";
      }
    }, [x, y, isOver, node]);

    return (
      (isOver && node) && (
        <div className="tooltip" ref={ref}>
          <div>{path}</div>
          <pre>
            {content}
          </pre>
        </div>
      )
    );
  };
