import {loadData} from "./utils.ts";
import {useCallback, useEffect, useMemo, useState} from "react";
import {Entry} from "./entry.ts";
import {useWindowSize} from "usehooks-ts";
import {hierarchy, HierarchyNode, HierarchyRectangularNode, treemap, treemapResquarify} from "d3-hierarchy";
import {group} from "d3-array";
import createRainbowColor from "./color.ts";
import {Tooltip} from "./Tooltip.tsx";
import {Node} from "./Node.tsx";

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
            .paddingOuter(2)
            .paddingTop(20)
            .round(true)
            .tile(treemapResquarify);
    }, [height, width])

    const [selectedNode, setSelectedNode] = useState<HierarchyRectangularNode<Entry> | null>(null)

    const getZoomMultiplier = useCallback((node: Entry) => {
        if (selectedNode === null) {
            return 1
        }

        const leaves = new Set(selectedNode.leaves().map((d) => d.data))
        return leaves.has(node) ? 1 : 0
    }, [selectedNode])

    const [showTooltip, setShowTooltip] = useState<boolean>(false);
    const [tooltipNode, setTooltipNode] = useState<
        HierarchyRectangularNode<Entry> | undefined
    >(undefined);

    useEffect(() => {
        const handleMouseOut = () => {
            setShowTooltip(false);
        };

        document.addEventListener("mouseover", handleMouseOut);
        return () => {
            document.removeEventListener("mouseover", handleMouseOut);
        };
    }, []);

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
                                        onMouseOver={(node) => {
                                            setTooltipNode(node);
                                            setShowTooltip(true);
                                        }}
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
