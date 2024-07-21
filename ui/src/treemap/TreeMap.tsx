import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { group } from "d3-array";
import type { HierarchyNode } from "d3-hierarchy";
import { hierarchy, treemap, treemapSquarify } from "d3-hierarchy";
import { useTitle, useWindowSize } from "react-use";
import { Group, Layer, Stage } from "react-konva";
import type Konva from "konva";
import type { Entry } from "../tool/entry.ts";
import createRainbowColor from "../tool/color.ts";
import { Tooltip } from "../Tooltip.tsx";
import { Node } from "../Node.tsx";
import "./style.scss";
import { trimPrefix } from "../tool/utils.ts";
import { shallowCopy } from "../tool/copy.ts";
import type { TreeMapProps } from "./props.ts";

export default function TreeMap({ entry }: TreeMapProps) {
  // Set the document title to the name of the entry
  useTitle(entry.getName(), {
    restoreOnUnmount: true,
  });

  // Get the window size
  const { width, height } = useWindowSize();

  const rawHierarchy = useMemo(() => {
    return hierarchy(entry, e => e.getChildren())
      .sum((e) => {
        if (e.getChildren().length === 0) {
          return e.getSize();
        }
        return 0;
      })
      .sort((a, b) => a.data.getSize() - b.data.getSize());
  }, [entry]);

  const rawHierarchyID = useMemo(() => {
    const cache = new Map<number, HierarchyNode<Entry>>();
    rawHierarchy.descendants().forEach((node) => {
      cache.set(node.data.getID(), node);
    });
    return cache;
  }, [rawHierarchy]);

  const getModuleColorRaw = useMemo(() => {
    return createRainbowColor(rawHierarchy);
  }, [rawHierarchy]);

  const getModuleColor = useCallback((id: number) => {
    return getModuleColorRaw(rawHierarchyID.get(id)!);
  }, [getModuleColorRaw, rawHierarchyID]);

  const layout = useMemo(() => {
    return treemap<Entry>()
      .size([width, height])
      .paddingInner(1)
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

    return cur.data.getID();
  }, [entry, rawHierarchy]);

  const [selectedNodeID, setSelectedNodeID] = useState<number | null>(loadNodeFromHash);

  const layoutRoot = useMemo(() => {
    let root: HierarchyNode<Entry> | null;
    if (!selectedNodeID || selectedNodeID === rawHierarchy.data.getID()) {
      root = rawHierarchy;
    }
    else {
      const selectedNode = rawHierarchyID.get(selectedNodeID)!;
      const ancestors = selectedNode.ancestors().reverse();

      function writeValue(n: HierarchyNode<Entry>, value?: number) {
        // @ts-expect-error write to readonly
        // noinspection JSConstantReassignment
        n.value = value;
      }

      root = shallowCopy(rawHierarchy);
      writeValue(root, selectedNode.value);

      let cur = root;
      for (let i = 1; i < ancestors.length; i++) {
        // use shallowCopy
        const node = shallowCopy(ancestors[i]);
        writeValue(node, selectedNode.value);
        cur.children = [node];
        cur = node;
      }
    }

    return layout(root!);
  }, [layout, rawHierarchy, rawHierarchyID, selectedNodeID]);

  const layers = useMemo(() => {
    const layerMap = group(
      layoutRoot.descendants(),
      (d: HierarchyNode<Entry>) => d.height,
    );
    const layerArray = Array.from(layerMap, ([key, values]) => ({
      key,
      values,
    }));
    layerArray.sort((a, b) => b.key - a.key);
    return layerArray;
  }, [layoutRoot]);

  useEffect(() => {
    if (selectedNodeID === null) {
      if (location.hash !== "") {
        history.replaceState(null, "", " ");
      }
      return;
    }
    const selectedNode = rawHierarchyID.get(selectedNodeID);
    if (!selectedNode) {
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
  }, [rawHierarchyID, selectedNodeID]);

  const stageRef = useRef<Konva.Stage>(null);

  const getTargetNode = useCallback((e: Konva.KonvaEventObject<MouseEvent>) => {
    console.log(e);

    const dataId = e.target._id;

    return rawHierarchyID.get(dataId)?.data ?? null;
  }, [rawHierarchyID]);

  const onClick = useCallback((e: Konva.KonvaEventObject<MouseEvent>) => {
    const node = getTargetNode(e);
    if (node === null) {
      console.log("no node");
      return;
    }

    if (e.evt.ctrlKey) {
      console.log(node);
      return;
    }

    if (selectedNodeID === node.getID()) {
      setSelectedNodeID(null);
    }
    else {
      setSelectedNodeID(node.getID());
    }
  }, [getTargetNode, selectedNodeID]);

  const nodes = useMemo(() => {
    return layers.map(({ key, values }) => {
      return (
        <Group key={key}>
          {values.map((node) => {
            if (node.x1 - node.x0 < 2 || node.y1 - node.y0 < 2) {
              return null;
            }

            const { backgroundColor, fontColor } = getModuleColor(node.data.getID());

            return (
              <Node
                key={node.data.getID()}
                id={node.data.getID()}
                title={node.data.getName()}
                selected={selectedNodeID === node.data.getID()}
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
        </Group>
      );
    });
  }, [getModuleColor, layers, selectedNodeID]);

  return (
    <>
      <Tooltip
        moveRef={stageRef}
        getTargetNode={getTargetNode}
      />
      <Stage
        width={width}
        height={height}
        onClick={onClick}
        ref={stageRef}
      >
        <Layer>
          {nodes}
        </Layer>
      </Stage>
    </>
  );
}
