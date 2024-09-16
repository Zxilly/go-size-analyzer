import type { RefObject } from "react";
import type { Entry } from "./tool/entry.ts";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { useMouse } from "./tool/useMouse.ts";

const Tooltip_marginX = 10;
const Tooltip_marginY = 30;

export interface TooltipProps {
  moveRef: RefObject<SVGElement>;
  getTargetNode: (e: EventTarget) => Entry | null;
}

export const Tooltip: React.FC<TooltipProps>
    = ({
      moveRef,
      getTargetNode,
    }) => {
      const ref = useRef<HTMLDivElement>(null);
      const [style, setStyle] = useState<React.CSSProperties>({
        visibility: "hidden",
      });

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

      useEffect(() => {
        if (!ref.current || !x || !y || !isOver) {
          return;
        }

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

        setStyle(pos);
      }, [isOver, x, y]);

      return (
        (isOver && node) && (
          <div className="tooltip" ref={ref} style={style}>
            <div>{path}</div>
            <pre>
              {content}
            </pre>
          </div>
        )
      );
    };
