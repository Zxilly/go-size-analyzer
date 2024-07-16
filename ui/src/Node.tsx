import React, { useLayoutEffect, useMemo, useRef } from "react";
import { PADDING, TOP_PADDING } from "./tool/const.ts";

export interface NodeProps {
  id: number;
  title: string;
  selected: boolean;

  x0: number;
  x1: number;
  y0: number;
  y1: number;
  hasChildren: boolean;

  backgroundColor: string;
  fontColor: string;
}

export const Node: React.FC<NodeProps> = React.memo((
  {
    id,
    title,
    selected,

    x0,
    x1,
    y0,
    y1,
    hasChildren,

    backgroundColor,
    fontColor,
  },
) => {
  const textRef = useRef<SVGTextElement>(null);
  const textRectRef = useRef<DOMRect | null>(null);

  const width = x1 - x0;
  const height = y1 - y0;

  const textProps = useMemo(() => {
    return {
      fontSize: "0.8em",
      dominantBaseline: "middle",
      textAnchor: "middle",
      x: width / 2,
      y: hasChildren ? (TOP_PADDING + PADDING) / 2 : height / 2,
    };
  }, [hasChildren, height, width]);

  useLayoutEffect(() => {
    if (width === 0 || height === 0 || !textRef.current) {
      return;
    }

    if (textRectRef.current == null) {
      textRectRef.current = textRef.current.getBoundingClientRect();
    }

    let scale: number;
    if (hasChildren) {
      scale = Math.min(
        (width * 0.9) / textRectRef.current.width,
        Math.min(height, TOP_PADDING + PADDING) / textRectRef.current.height,
      );
      scale = Math.min(1, scale);
      textRef.current.setAttribute(
        "y",
        String(Math.min(TOP_PADDING + PADDING, height) / 2 / scale),
      );
      textRef.current.setAttribute("x", String(width / 2 / scale));
    }
    else {
      scale = Math.min(
        (width * 0.9) / textRectRef.current.width,
        (height * 0.9) / textRectRef.current.height,
      );
      scale = Math.min(1, scale);
      textRef.current.setAttribute("y", String(height / 2 / scale));
      textRef.current.setAttribute("x", String(width / 2 / scale));
    }

    textRef.current.setAttribute("transform", `scale(${scale.toFixed(2)})`);
  }, [hasChildren, height, width]);

  return (
    <g
      className="node"
      transform={`translate(${x0},${y0})`}
      data-id={id}
    >
      <rect
        fill={backgroundColor}
        rx={2}
        ry={2}
        width={x1 - x0}
        height={y1 - y0}
        stroke={selected ? "#fff" : undefined}
        strokeWidth={selected ? 2 : undefined}
      >
      </rect>
      <text
        ref={textRef}
        fill={fontColor}
        onClick={(event) => {
          if (window.getSelection()?.toString() !== "") {
            event.stopPropagation();
          }
        }}
        {...textProps}
      >
        {title}
      </text>
    </g>
  );
});
