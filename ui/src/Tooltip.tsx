import React, {useEffect, useMemo, useRef, useState} from "react";
import {HierarchyRectangularNode} from "d3-hierarchy";
import {Entry} from "./entry.ts";

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

        const content = useMemo(() => {
            if (!node) return null;

            const path = node
                .ancestors()
                .reverse()
                .map((d) => d.data.getName())
                .join("/");

            return (
                <>
                    <div>{path}</div>
                    <pre>
                        {node.data.toString()}
                    </pre>
                </>
            );
        }, [node]);

        const updatePosition = (mouseCoords: { x: number; y: number }) => {
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
        };

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
                {content}
            </div>
        );
    };
