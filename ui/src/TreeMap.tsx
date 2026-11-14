import type { Entry } from "./tool/entry.ts";
import { group } from "d3-array";
import * as React from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTitle, useWindowSize } from "react-use";
import { FunctionDetails } from "./FunctionDetails.tsx";
import { ItemBrowser } from "./ItemBrowser.tsx";
import { Node } from "./Node.tsx";
import createRainbowColor from "./tool/color.ts";
import { FileImpl, FunctionImpl } from "./tool/entry.ts";
import { buildViewportHierarchy, pathFromHash } from "./tool/viewport.ts";
import { Tooltip } from "./Tooltip.tsx";
import "./style.scss";

interface TreeMapProps {
  entry: Entry;
}

function TreeMapView({ entry }: TreeMapProps) {
  useTitle(entry.getName(), { restoreOnUnmount: true });
  const { width, height } = useWindowSize();
  const [selection, setSelection] = useState<Entry[] | null>(() => pathFromHash(entry, location.hash));
  const selectedEntry = selection?.[selection.length - 1];
  const selectedNodeID = selectedEntry?.getID() ?? null;
  const view = useMemo(() => buildViewportHierarchy(entry, width, height, selection), [entry, width, height, selection]);
  const getModuleColor = useMemo(() => createRainbowColor(view.root), [view]);
  const layers = useMemo(() => [
    ...group(view.root.descendants(), node => node.depth),
  ].sort(([a], [b]) => a - b), [view]);

  useEffect(() => {
    const restore = () => setSelection(pathFromHash(entry, location.hash));
    window.addEventListener("hashchange", restore);
    return () => window.removeEventListener("hashchange", restore);
  }, [entry]);

  useEffect(() => {
    const path = selection
      ? `#${selection.map(item => item.getURLSafeName()).join("#")}`
      : "";
    if (location.hash !== path) {
      history.replaceState(null, "", path || `${location.pathname}${location.search}`);
    }
  }, [selection]);

  const svgRef = useRef<SVGSVGElement>(null);
  const getTargetNode = useCallback((target: EventTarget) => {
    const element = (target as Element).closest?.(".node[data-id]");
    const id = element?.getAttribute("data-id");
    return id
      ? view.byID.get(Number(id))?.data ?? null
      : null;
  }, [view]);

  const onClick = useCallback((event: React.MouseEvent<SVGSVGElement>) => {
    const node = getTargetNode(event.target);
    if (!node) {
      return;
    }
    if (event.ctrlKey) {
      console.log(node);
      return;
    }
    setSelection(selectedNodeID === node.getID()
      ? null
      : view.byID.get(node.getID())!.ancestors().reverse().map(item => item.data));
  }, [getTargetNode, selectedNodeID, view]);

  return (
    <>
      {!(selectedEntry instanceof FunctionImpl) && <Tooltip moveRef={svgRef} getTargetNode={getTargetNode} />}
      <svg
        id="treemap-svg"
        xmlns="http://www.w3.org/2000/svg"
        viewBox={`0 0 ${width} ${height}`}
        onClick={onClick}
        ref={svgRef}
      >
        {layers.map(([depth, nodes]) => (
          <g className="layer" key={depth}>
            {nodes.map((node) => {
              if (node.x1 - node.x0 < 2 || node.y1 - node.y0 < 2) {
                return null;
              }
              const { backgroundColor, fontColor } = getModuleColor(node);
              const type = node.data.getType();
              const expandable = type === "package" || type === "container" || type === "result"
                || (node.data instanceof FileImpl && node.data.hasFunctions());
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
                  hasChildren={expandable}
                />
              );
            })}
          </g>
        ))}
      </svg>
      {selectedEntry instanceof FunctionImpl && (
        <FunctionDetails
          entry={selectedEntry}
          onClose={() => setSelection(selection!.slice(0, -1))}
        />
      )}
      {selectedEntry && !(selectedEntry instanceof FunctionImpl) && selectedEntry.getChildren().length > 0 && (
        <ItemBrowser
          key={selectedEntry.getID()}
          entry={selectedEntry}
          onSelect={child => setSelection([...selection!, child])}
        />
      )}
    </>
  );
}

function TreeMap({ entry }: TreeMapProps) {
  // Explorer can replace the report without unmounting this component.
  // Paths contain Entry objects and must belong to the current report.
  return <TreeMapView key={entry.getID()} entry={entry} />;
}

export default TreeMap;
