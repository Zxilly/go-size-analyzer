import React, {useCallback, useEffect, useMemo, useRef, useState} from "react";
import {HierarchyRectangularNode} from "d3-hierarchy";
import {Entry} from "./tool/entry.ts";

const Tooltip_marginX = 10;
const Tooltip_marginY = 30;

export interface TooltipProps {
    node?: HierarchyRectangularNode<Entry>;
    visible: boolean;
}

export const Tooltip: React.FC<TooltipProps> =
    ({
         node,
         visible,
     }) => {
        const ref = useRef<HTMLDivElement>(null);
        const [style, setStyle] = useState({});

        const path = useMemo(() => {
            if (!node) return "";

            return node
                .ancestors()
                .reverse()
                .slice(1)
                .map((d) => d.data.getName())
                .join("/");
        }, [node])

        const content = useMemo(() => {
            return node?.data.toString() ?? "";
        }, [node]);

        const updatePosition = useCallback((mouseCoords: { x: number; y: number }) => {
            if (!ref.current) return;

            const pos = {
                left: mouseCoords.x + Tooltip_marginX,
                top: mouseCoords.y + Tooltip_marginY,
            };

            const boundingRect = ref.current.getBoundingClientRect();

            if (pos.left + boundingRect.width > window.innerWidth) {
                // Shifting horizontally
                pos.left = window.innerWidth - boundingRect.width;
            }

            if (pos.top + boundingRect.height > window.innerHeight) {
                // Flipping vertically
                pos.top = mouseCoords.y - Tooltip_marginY - boundingRect.height;
            }

            setStyle(pos);
        }, []);

        useEffect(() => {
            const handleMouseMove = (event: MouseEvent) => {
                updatePosition({
                    x: event.pageX,
                    y: event.pageY,
                });
            };

            document.addEventListener("mousemove", handleMouseMove, true);
            return () => {
                document.removeEventListener("mousemove", handleMouseMove, true);
            };
        }, []);

        return (
            <div className={`tooltip ${visible ? "" : "tooltip-hidden"}`} ref={ref} style={style}>
                <div>{path}</div>
                <pre>
                    {content}
                </pre>
            </div>
        );
    };
