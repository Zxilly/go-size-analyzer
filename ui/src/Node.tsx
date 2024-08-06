import React, { useCallback, useMemo, useRef } from "react";
import memoize from "lodash.memoize";
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

let textElement: SVGTextElement;

(function init() {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.style.position = "absolute";
  svg.style.visibility = "hidden";
  document.body.appendChild(svg);
  textElement = document.createElementNS("http://www.w3.org/2000/svg", "text");
  textElement.setAttribute("font-size", "0.8em");
  textElement.setAttribute("dominant-baseline", "middle");
  textElement.setAttribute("text-anchor", "middle");
  svg.appendChild(textElement);
})();

function measureText(text: string): [number, number] {
  textElement.textContent = text;
  const rect = textElement.getBoundingClientRect();

  return [rect.width, rect.height];
}

const memoizedMeasureText = memoize(measureText);

interface RenderAttributes {
  x: number;
  y: number;
  scale: number;
  display: string;
}

function getTransform(scale: number): string {
  return `scale(${scale.toFixed(2)})`;
}

const splitter = /[/\\]/;

function getLastWord(title: string): string {
  const words = title.split(splitter);
  return words[words.length - 1];
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

  const getScale = useCallback((title: string, fallback: boolean = true): [string, number] => {
    if (title === "") {
      return ["", 0];
    }

    const [textWidth, textHeight] = memoizedMeasureText(title);

    let scale: number;
    if (hasChildren) {
      scale = Math.min(
        (width * 0.9) / textWidth,
        Math.min(height, TOP_PADDING + PADDING) / textHeight,
      );
      scale = Math.min(1, scale);
    }
    else {
      scale = Math.min(
        (width * 0.9) / textWidth,
        (height * 0.9) / textHeight,
      );
      if (scale > 1) {
        scale = Math.sqrt(scale);
      }
    }

    if (scale < 0.7 && fallback) {
      return getScale(getLastWord(title), false);
    }
    return [title, scale];
  }, [hasChildren, height, width]);

  const renderAttr = useMemo<RenderAttributes>(() => {
    const [display, scale] = getScale(title);
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
  }, [getScale, title, hasChildren, width, height]);

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
