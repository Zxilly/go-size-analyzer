import {Entry} from "./tool/entry.ts";
import {HierarchyRectangularNode} from "d3-hierarchy";

export const globalNodeCache = new Map<number, HierarchyRectangularNode<Entry>>
