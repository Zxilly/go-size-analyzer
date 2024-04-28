import {HierarchyRectangularNode} from "d3-hierarchy";
import React, {useLayoutEffect, useMemo, useRef} from "react";
import {Entry} from "./tool/entry.ts";
import {NodeColorGetter} from "./tool/color.ts";
import {PADDING, TOP_PADDING} from "./tool/const.ts";
import {trimPrefix} from "./tool/utils.ts";

type NodeEventHandler = (event: HierarchyRectangularNode<Entry>) => void;

export interface NodeProps {
    node: HierarchyRectangularNode<Entry>;
    onMouseOver: NodeEventHandler;
    selected: boolean;
    onClick: NodeEventHandler;
    getModuleColor: NodeColorGetter;
}

export const Node: React.FC<NodeProps> = (
    {
        node,
        onMouseOver,
        onClick,
        selected,
        getModuleColor
    }
) => {
    const {backgroundColor, fontColor} = getModuleColor(node);
    const {x0, x1, y1, y0, children = null} = node;

    const textRef = useRef<SVGTextElement>(null);
    const textRectRef = useRef<DOMRect>();

    const width = x1 - x0;
    const height = y1 - y0;

    const textProps = useMemo(() => {
        return {
            "fontSize": "0.9em",
            "dominantBaseline": "middle",
            "textAnchor": "middle",
            x: width / 2,
            y: children != null ? (TOP_PADDING + PADDING) / 2 : height / 2
        }
    }, [])

    const title = useMemo(() => {
        const t = trimPrefix(node.data.getName(), node.parent?.data.getName() ?? "")
        return trimPrefix(t, "/")
    }, [node.data, node.parent?.data])

    useLayoutEffect(() => {
        if (width == 0 || height == 0 || !textRef.current) {
            return;
        }

        if (textRectRef.current == null) {
            textRectRef.current = textRef.current.getBoundingClientRect();
        }

        let scale: number;
        if (children != null) {
            scale = Math.min(
                (width * 0.9) / textRectRef.current.width,
                Math.min(height, TOP_PADDING + PADDING) / textRectRef.current.height
            );
            scale = Math.min(1, scale);
            textRef.current.setAttribute(
                "y",
                String(Math.min(TOP_PADDING + PADDING, height) / 2 / scale)
            );
            textRef.current.setAttribute("x", String(width / 2 / scale));
        } else {
            scale = Math.min(
                (width * 0.9) / textRectRef.current.width,
                (height * 0.9) / textRectRef.current.height
            );
            scale = Math.min(1, scale);
            textRef.current.setAttribute("y", String(height / 2 / scale));
            textRef.current.setAttribute("x", String(width / 2 / scale));
        }

        textRef.current.setAttribute("transform", `scale(${scale.toFixed(2)})`);
    }, [children, height, width]);

    if (width == 0 || height == 0) {
        return null;
    }

    return (
        <g
            className="node"
            transform={`translate(${x0},${y0})`}
            onClick={(event) => {
                event.stopPropagation();
                onClick(node);
            }}
            onMouseOver={(event) => {
                event.stopPropagation();
                onMouseOver(node);
            }}
        >
            <rect
                fill={backgroundColor}
                rx={2}
                ry={2}
                width={x1 - x0}
                height={y1 - y0}
                stroke={selected ? "#fff" : undefined}
                strokeWidth={selected ? 2 : undefined}
            ></rect>
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
};
