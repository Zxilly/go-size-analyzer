import React, { useCallback, useEffect, useMemo, useState } from "react";
import { group } from "d3-array";
import type { HierarchyNode, HierarchyRectangularNode } from "d3-hierarchy";
import { hierarchy, treemap, treemapSquarify } from "d3-hierarchy";
import { useTitle, useWindowSize } from "react-use";

import type { Entry } from "./tool/entry.ts";
import createRainbowColor from "./tool/color.ts";
import { Tooltip } from "./Tooltip.tsx";
import { Node } from "./Node.tsx";

import "./style.scss";
import { trimPrefix } from "./tool/utils.ts";

interface TreeMapProps {
  entry: Entry;
}

function TreeMap({ entry }: TreeMapProps) {
  // Set the document title to the name of the entry
  useTitle(entry.getName(), {
    restoreOnUnmount: true,
  });

  // Get the window size
  const { width, height } = useWindowSize();

  const rawHierarchy = useMemo(() => {
    return hierarchy(entry, e => e.getChildren());
  }, [entry]);

  const getModuleColor = useMemo(() => {
    return createRainbowColor(rawHierarchy);
  }, [rawHierarchy]);

  const layout = useMemo(() => {
    return treemap<Entry>()
      .size([width, height])
      .paddingInner(2)
      .paddingTop(20)
      .round(true)
      .tile(treemapSquarify);
  }, [height, width]);

  const loadNodeFromHash = useCallback(() => {
    const parts = trimPrefix(location.hash, "#").split("#");
    if (parts.length >= 1) {
      const base = parts[0];
      if (base !== entry.getURLSafeName()) {
        return null;
      }
    }
    let cur = rawHierarchy;
    for (let i = 1; i < parts.length; i++) {
      const part = parts[i];
      if (!cur.children) {
        return null;
      }

      const found = cur.children.find(d => d.data.getURLSafeName() === part);
      if (!found) {
        return null;
      }
      cur = found;
    }

    return cur;
  }, [entry, rawHierarchy]);

  const [selectedNode, setSelectedNode] = useState<HierarchyNode<Entry> | null>(loadNodeFromHash);
  const selectedNodeLeaveIDSet = useMemo(() => {
    if (selectedNode === null) {
      return new Set<number>();
    }

    return new Set(selectedNode.leaves().map(d => d.data.getID()));
  }, [selectedNode]);

  const root = useMemo(() => {
    const rootWithSizesAndSorted = rawHierarchy
      .sum((node) => {
        if (node.getChildren().length === 0) {
          if (selectedNode) {
            if (!selectedNodeLeaveIDSet.has(node.getID())) {
              return 0;
            }
          }

          return node.getSize();
        }
        return 0;
      })
      .sort((a, b) => a.data.getSize() - b.data.getSize());

    return layout(rootWithSizesAndSorted);
  }, [layout, rawHierarchy, selectedNode, selectedNodeLeaveIDSet]);

  const nestedData = useMemo(() => {
    const nestedDataMap = group(
      root.descendants(),
      (d: HierarchyNode<Entry>) => d.height,
    );
    const nested = Array.from(nestedDataMap, ([key, values]) => ({
      key,
      values,
    }));
    nested.sort((a, b) => b.key - a.key);
    return nested;
  }, [root]);

  const allNodes = useMemo(() => {
    const cache = new Map<number, HierarchyRectangularNode<Entry>>();
    root.descendants().forEach((node) => {
      cache.set(node.data.getID(), node);
    });
    return cache;
  }, [root]);

  useEffect(() => {
    if (selectedNode === null) {
      if (location.hash !== "") {
        history.replaceState(null, "", " ");
      }
      return;
    }

    const path = `#${selectedNode
      .ancestors()
      .map((d) => {
        return d.data.getURLSafeName();
      })
      .reverse()
      .join("#")}`;

    if (location.hash !== path) {
      history.replaceState(null, "", path);
    }
  }, [selectedNode]);

  const [showTooltip, setShowTooltip] = useState(false);
  const [tooltipPosition, setTooltipPosition] = useState<[number, number]>([0, 0]);
  const [tooltipNode, setTooltipNode]
        = useState<HierarchyRectangularNode<Entry> | undefined>(undefined);

  const onMouseEnter = useCallback(() => {
    setShowTooltip(true);
  }, []);

  const onMouseLeave = useCallback(() => {
    setShowTooltip(false);
  }, []);

  const getTargetNode = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    if (!e.target) {
      return null;
    }

    const target = (e.target as SVGElement).parentNode;
    if (!target) {
      return null;
    }

    const dataIdStr = (target as Element).getAttribute("data-id");
    if (!dataIdStr) {
      return null;
    }

    const dataId = Number.parseInt(dataIdStr);

    return allNodes.get(dataId) ?? null;
  }, [allNodes]);

  const onMouseMove = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    setTooltipPosition([e.clientX, e.clientY]);

    const node = getTargetNode(e);
    if (node === null) {
      setTooltipNode(undefined);
      return;
    }

    setTooltipNode(node);
  }, [getTargetNode]);

  const onClick = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    const node = getTargetNode(e);
    if (node === null) {
      return;
    }

    if (selectedNode?.data.getID() === node.data.getID()) {
      setSelectedNode(null);
    }
    else {
      setSelectedNode(node);
    }
  }, [getTargetNode, selectedNode]);

  const nodes = useMemo(() => {
    const selectedID = selectedNode?.data.getID();

    return (
      nestedData.map(({ key, values }) => {
        return (
          <g className="layer" key={key}>
            {values.map((node) => {
              const { backgroundColor, fontColor } = getModuleColor(node);

              if (node.x0 === node.x1 || node.y0 === node.y1) {
                return null;
              }

              return (
                <Node
                  key={node.data.getID()}
                  id={node.data.getID()}
                  title={node.data.getName()}
                  selected={selectedID === node.data.getID()}
                  x0={node.x0}
                  y0={node.y0}
                  x1={node.x1}
                  y1={node.y1}

                  backgroundColor={backgroundColor}
                  fontColor={fontColor}
                  hasChildren={node.children !== undefined}
                />
              );
            }).filter(Boolean)}
          </g>
        );
      })
    );
  }, [getModuleColor, nestedData, selectedNode]);

  const tooltipVisible = useMemo(() => {
    return showTooltip && tooltipNode && tooltipPosition.every(v => v > 0);
  }, [showTooltip, tooltipNode, tooltipPosition]);

  return (
    <>
      {(tooltipVisible) && <Tooltip node={tooltipNode!.data} x={tooltipPosition[0]} y={tooltipPosition[1]} />}
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox={`0 0 ${width} ${height}`}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        onMouseMoveCapture={onMouseMove}
        onClick={onClick}
      >
        {nodes}
      </svg>
    </>
  );
}

export default TreeMap;
