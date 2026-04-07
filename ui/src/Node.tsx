import * as React from "react";
import { useMemo, useRef } from "react";
import { PADDING, TOP_PADDING } from "./tool/const.ts";
import { getScale } from "./tool/measureText.ts";

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

interface RenderAttributes {
  x: number;
  y: number;
  scale: number;
  display: string;
}

function getTransform(scale: number): string | undefined {
  if (scale === 1) {
    return undefined;
  }

  return `scale(${scale.toFixed(2)})`;
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

  const width = x1 - x0;
  const height = y1 - y0;

  const renderAttr = useMemo<RenderAttributes>(() => {
    const [display, scale] = getScale(title, width, height, hasChildren);
    if (hasChildren) {
      return {
        x: width / 2 / scale,
        y: Math.min(TOP_PADDING + PADDING, height) / 2 / scale,
        display,
        scale,
      };
    }
    else {
      return {
        x: width / 2 / scale,
        y: height / 2 / scale,
        display,
        scale,
      };
    }
  }, [title, width, height, hasChildren]);

  return (
    <g
      className="node"
      transform={`translate(${x0},${y0})`}
      data-id={id}
    >
      <rect
        fill={backgroundColor}
        width={width}
        height={height}
        stroke={selected ? "#fff" : undefined}
        strokeWidth={selected ? 2 : undefined}
      >
      </rect>
      {
        width > 12 && height > 12 && renderAttr.scale > 0.5 && (
          <text
            ref={textRef}
            fill={fontColor}
            onClick={(event) => {
              if (window.getSelection()?.toString() !== "") {
                event.stopPropagation();
              }
            }}
            fontSize="0.8em"
            dominantBaseline="middle"
            textAnchor="middle"
            x={renderAttr.x}
            y={renderAttr.y}
            transform={getTransform(renderAttr.scale)}
          >
            {renderAttr.display}
          </text>
        )
      }
    </g>
  );
});
