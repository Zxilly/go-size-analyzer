import {loadData} from "./tool/utils.ts";
import {useCallback, useEffect, useMemo, useRef, useState} from "react";
import {Entry} from "./tool/entry.ts";
import {useWindowSize} from "usehooks-ts";
import {hierarchy, HierarchyNode, HierarchyRectangularNode, treemap, treemapResquarify} from "d3-hierarchy";
import {group} from "d3-array";
import createRainbowColor from "./tool/color.ts";
import {Tooltip} from "./Tooltip.tsx";
import {Node} from "./Node.tsx";
import {globalNodeCache} from "./cache.ts";

function TreeMap() {
    const rootEntry = useMemo(() => new Entry(loadData()), [])

    // Set the document title to the name of the entry
    useEffect(() => {
        document.title = rootEntry.getName()
    }, [rootEntry])

    // Get the window size
    const {width, height} = useWindowSize()

    const rawHierarchy = useMemo(() => {
        return hierarchy(rootEntry, (e) => e.getChildren())
    }, [rootEntry])

    const getModuleColor = useMemo(() => {
        return createRainbowColor(rawHierarchy)
    }, [rawHierarchy])

    const layout = useMemo(() => {
        return treemap<Entry>()
            .size([width, height])
            .paddingInner(2)
            //.paddingOuter(2)
            .paddingTop(20)
            .round(true)
            .tile(treemapResquarify.ratio(1));
    }, [height, width])

    const [selectedNode, setSelectedNode] = useState<HierarchyRectangularNode<Entry> | null>(null)
    const selectedNodeLeaveSet = useMemo(() => {
        if (selectedNode === null) {
            return new Set<Entry>()
        }

        return new Set(selectedNode.leaves().map((d) => d.data))
    }, [selectedNode])

    const getZoomMultiplier = useCallback((node: Entry) => {
        if (selectedNode === null) {
            return 1
        }

        return selectedNodeLeaveSet.has(node) ? 1 : 0
    }, [selectedNode, selectedNodeLeaveSet])


    const root = useMemo(() => {
        const rootWithSizesAndSorted = rawHierarchy
            .sum((node) => {
                const zoom = getZoomMultiplier(node)
                if (zoom === 0) {
                    return 0
                }

                if (node.getChildren().length === 0) {
                    return node.getSize()
                }
                return 0
            })
            .sort((a, b) => a.data.getSize() - b.data.getSize())
        return layout(rootWithSizesAndSorted)
    }, [getZoomMultiplier, layout, rawHierarchy])

    const nestedData = useMemo(() => {
        const nestedDataMap = group(
            root.descendants(),
            (d: HierarchyNode<Entry>) => d.height
        );
        const nestedData = Array.from(nestedDataMap, ([key, values]) => ({
            key,
            values,
        }));
        nestedData.sort((a, b) => b.key - a.key);
        return nestedData;
    }, [root]);

    const [showTooltip, setShowTooltip] = useState(false);
    const [tooltipNode, setTooltipNode] =
        useState<HierarchyRectangularNode<Entry> | undefined>(undefined);

    const svgRef = useRef<SVGSVGElement>(null);

    useEffect(() => {
        if (!svgRef.current) {
            return;
        }
        const svg = svgRef.current;

        const visibleListener = (value: boolean) => {
            return () => {
                setShowTooltip(value);
            }
        }
        const enter = visibleListener(true);
        const leave = visibleListener(false);

        svg.addEventListener("mouseenter", enter);
        svg.addEventListener("mouseleave", leave);

        return () => {
            svg.removeEventListener("mouseenter", enter);
            svg.removeEventListener("mouseleave", leave);
        }
    }, []);

    useEffect(() => {
        const moveListener = (e: MouseEvent) => {
            if (!e.target) {
                return;
            }

            const target = (e.target as SVGElement).parentNode;
            if (!target) {
                return;
            }

            const dataIdStr = (target as Element).getAttribute("data-id");
            if (!dataIdStr) {
                return;
            }

            const dataId = parseInt(dataIdStr);

            const node = globalNodeCache.get(dataId);
            if (!node) {
                return;
            }

            setTooltipNode(node);
        }

        document.addEventListener("mousemove", moveListener);
        return () => {
            document.removeEventListener("mousemove", moveListener);
        }
    }, []);

    return (
        <>
            <Tooltip visible={showTooltip} node={tooltipNode}/>
            <svg xmlns="http://www.w3.org/2000/svg" viewBox={`0 0 ${width} ${height}`}>
                {nestedData.map(({key, values}) => {
                    return (
                        <g className="layer" key={key}>
                            {values.map((node) => {
                                return (
                                    <Node
                                        key={node.data.getID()}
                                        node={node}
                                        selected={selectedNode?.data?.getID() === node.data.getID()}
                                        onClick={(node) => {
                                            setSelectedNode(selectedNode?.data?.getID() === node.data.getID() ? null : node);
                                        }}
                                        getModuleColor={getModuleColor}
                                    />
                                );
                            })}
                        </g>
                    );
                })}
            </svg>
        </>
    )
}

export default TreeMap
