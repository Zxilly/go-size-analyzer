import React, { useMemo, useRef } from "react";
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

const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
svg.style.position = "absolute";
svg.style.visibility = "hidden";
document.body.appendChild(svg);
const textElement = document.createElementNS("http://www.w3.org/2000/svg", "text");
textElement.setAttribute("font-size", "0.8em");
textElement.setAttribute("dominant-baseline", "middle");
textElement.setAttribute("text-anchor", "middle");
svg.appendChild(textElement);

function measureText(text: string): [number, number] {
  textElement.textContent = text;
  const rect = textElement.getBBox();

  return [rect.width, rect.height];
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

  const textProps = useMemo<Record<string, string | number>>(() => {
    const initial: Record<string, string | number> = {
      fontSize: "0.8em",
      dominantBaseline: "middle",
      textAnchor: "middle",
    };
    const [textWidth, textHeight] = measureText(title);
    let scale: number;
    if (hasChildren) {
      scale = Math.min(
        (width * 0.9) / textWidth,
        Math.min(height, TOP_PADDING + PADDING) / textHeight,
      );
      scale = Math.min(1, scale);
      initial.y = Math.min(TOP_PADDING + PADDING, height) / 2 / scale;
      initial.x = width / 2 / scale;
    }
    else {
      scale = Math.min(
        (width * 0.9) / textWidth,
        (height * 0.9) / textHeight,
      );
      scale = Math.min(1, scale);
      initial.y = height / 2 / scale;
      initial.x = width / 2 / scale;
    }

    initial.transform = `scale(${scale.toFixed(2)})`;

    return initial;
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
      {
        width > 12 && height > 12 && (
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
        )
      }
    </g>
  );
});
