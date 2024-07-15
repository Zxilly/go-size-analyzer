import React, { useEffect, useMemo, useRef, useState } from "react";
import type { Entry } from "./tool/entry.ts";

const Tooltip_marginX = 10;
const Tooltip_marginY = 30;

export interface TooltipProps {
  node: Entry;
  x: number;
  y: number;
}

export const Tooltip: React.FC<TooltipProps>
    = ({
      x,
      y,
      node,
    }) => {
      const ref = useRef<HTMLDivElement>(null);
      const [style, setStyle] = useState<React.CSSProperties>({
        visibility: "hidden",
      });

      const path = useMemo(() => {
        return node.getName();
      }, [node]);

      const content = useMemo(() => {
        return node.toString();
      }, [node]);

      useEffect(() => {
        if (!ref.current) {
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
      }, [x, y]);

      return (
        <div className="tooltip" ref={ref} style={style}>
          <div>{path}</div>
          <pre>
            {content}
          </pre>
        </div>
      );
    };
