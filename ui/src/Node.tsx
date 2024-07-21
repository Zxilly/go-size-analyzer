import React, { useMemo, useRef } from "react";
import memoize from "lodash.memoize";
import { Group, Rect, Text } from "react-konva";
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

let canvas: OffscreenCanvasRenderingContext2D;

(function init() {
  const offscreen = new OffscreenCanvas(256, 256);
  canvas = offscreen.getContext("2d")!;
})();

function measureText(text: string): [number, number] {
  const metrics = canvas.measureText(text);
  const width = metrics.width;
  const height = metrics.actualBoundingBoxAscent + metrics.actualBoundingBoxDescent;

  return [width, height];
}

const memoizedMeasureText = memoize(measureText);

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

  const [textPos, setTextPos] = useState({ x: 0, y: 0 });

  const textProps = useMemo<Record<string, string | number>>(() => {
    const [textWidth, textHeight] = memoizedMeasureText(title);
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
  }, [hasChildren, height, title, width]);

  return (
    // <g
    //   className="node"
    //   transform={`translate(${x0},${y0})`}
    //   data-id={id}
    // >
    //   <rect
    //     fill={backgroundColor}
    //     rx={2}
    //     ry={2}
    //     width={x1 - x0}
    //     height={y1 - y0}
    //     stroke={selected ? "#fff" : undefined}
    //     strokeWidth={selected ? 2 : undefined}
    //   >
    //   </rect>
    //   {
    //     width > 12 && height > 12 && (
    //       <text
    //         ref={textRef}
    //         fill={fontColor}
    //         {...textProps}
    //       >
    //         {title}
    //       </text>
    //     )
    //   }
    // </g>
    <Group
      id={String(id)}
      x={x0}
      y={y0}
    >
      <Rect
        fill={backgroundColor}
        height={height}
        width={width}
        stroke={selected ? "#fff" : undefined}
        strokeWidth={selected ? 2 : undefined}
      />
      <Text
        fill={fontColor}
        text={title}
        fontSize={13}
        align="center"
        verticalAlign="middle"
      />
    </Group>
  );
});
