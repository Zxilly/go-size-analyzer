import React, {useCallback, useEffect, useMemo, useRef, useState} from "react";
import {group} from "d3-array";
import {HierarchyNode, HierarchyRectangularNode, hierarchy, treemap, treemapSquarify} from "d3-hierarchy";
import {useTitle, useWindowSize} from "react-use";

import {Entry} from "./tool/entry.ts";
import createRainbowColor from "./tool/color.ts";
import {Tooltip} from "./Tooltip.tsx";
import {Node} from "./Node.tsx";

import "./style.scss"
import {trimPrefix} from "./tool/utils.ts";
import {useHash} from "./tool/useHash.ts";

interface TreeMapProps {
    entry: Entry
}

function TreeMap({entry}: TreeMapProps) {
    // Set the document title to the name of the entry
    useTitle(entry.getName(), {
        restoreOnUnmount: true
    })

    const [hash, setHash] = useHash()

    // Get the window size
    const {width, height} = useWindowSize()

    const rawHierarchy = useMemo(() => {
        return hierarchy(entry, (e) => e.getChildren())
    }, [entry])

    const getModuleColor = useMemo(() => {
        return createRainbowColor(rawHierarchy)
    }, [rawHierarchy])

    const layout = useMemo(() => {
        return treemap<Entry>()
            .size([width, height])
            .paddingInner(2)
            .paddingTop(20)
            .round(true)
            .tile(treemapSquarify);
    }, [height, width])

    const [selectedNode, setSelectedNode] = useState<HierarchyRectangularNode<Entry> | null>(null)
    const selectedNodeLeaveSet = useMemo(() => {
        if (selectedNode === null) {
            return new Set<number>()
        }

        return new Set(selectedNode.leaves().map((d) => d.data.getID()))
    }, [selectedNode])

    const getZoomMultiplier = useCallback((node: Entry) => {
        if (selectedNode === null) {
            return 1
        }

        return selectedNodeLeaveSet.has(node.getID()) ? 1 : 0
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

    const allNodes = useMemo(() => {
        const cache = new Map<number, HierarchyRectangularNode<Entry>>();
        root.descendants().forEach((node) => {
            cache.set(node.data.getID(), node);
        })
        return cache;
    }, [root])

    const setSelectedNodeWithHash = useCallback((node: HierarchyRectangularNode<Entry> | null) => {
        setSelectedNode(node)

        if (node === null) {
            setHash("")
            return
        }

        setHash(
            node
                .ancestors()
                .map((d) => {
                    return d.data.getURLSafeName()
                })
                .reverse()
                .join("#")
        )
    }, [setHash])

    useEffect(() => {
        const parts = trimPrefix(hash, "#").split("#")
        if (parts.length >= 1) {
            const base = parts[0]
            if (base !== entry.getURLSafeName()) {
                return
            }
        }
        let cur = root
        for (let i = 1; i < parts.length; i++) {
            const part = parts[i]
            if (!cur.children) {
                return;
            }

            const found = cur.children.find((d) => d.data.getURLSafeName() === part)
            if (!found) {
                return;
            }
            cur = found
        }

        setSelectedNode(cur)
    }, [hash, root, entry])

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

            const node = allNodes.get(dataId);
            if (!node) {
                return;
            }

            setTooltipNode(node);
        }

        document.addEventListener("mousemove", moveListener);
        return () => {
            document.removeEventListener("mousemove", moveListener);
        }
    }, [allNodes]);

    const nodes = useMemo(() => {
        return (
            <Nodes
                nestedData={nestedData}
                selectedNode={selectedNode}
                getModuleColor={getModuleColor}
                setSelectedNode={setSelectedNodeWithHash}
            />
        )
    }, [getModuleColor, nestedData, selectedNode, setSelectedNodeWithHash])

    return (
        <>
            <Tooltip visible={showTooltip} node={tooltipNode?.data}/>
            <svg xmlns="http://www.w3.org/2000/svg" viewBox={`0 0 ${width} ${height}`} ref={svgRef}>
                {nodes}
            </svg>
        </>
    )
}

interface NodesProps {
    nestedData: { key: number, values: HierarchyRectangularNode<Entry>[] }[]
    selectedNode: HierarchyRectangularNode<Entry> | null
    getModuleColor: (node: HierarchyNode<Entry>) => { backgroundColor: string, fontColor: string }
    setSelectedNode: (node: HierarchyRectangularNode<Entry> | null) => void
}

const Nodes: React.FC<NodesProps> =
    ({
         nestedData,
         selectedNode,
         getModuleColor,
         setSelectedNode
     }) => {
        const nodes = useMemo(() => {
            return (
                nestedData.map(({key, values}) => {
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
                })
            )
        }, [getModuleColor, nestedData, selectedNode?.data, setSelectedNode])

        return (
            <>
                {nodes}
            </>
        )
    }

export default TreeMap
