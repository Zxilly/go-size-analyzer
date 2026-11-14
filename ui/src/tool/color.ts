import type { RGBColor } from "d3-color";
import type { HierarchyNode } from "d3-hierarchy";
import type { Entry } from "./entry.ts";
import { hsl } from "d3-color";

import { scaleLinear, scaleSequential } from "d3-scale";

type CssColor = string;

export const COLOR_BASE: CssColor = "#cecece";

// https://www.w3.org/TR/WCAG20/#relativeluminancedef
const rc = 0.2126;
const gc = 0.7152;
const bc = 0.0722;
// low-gamma adjust coefficient
const lowc = 1 / 12.92;

function adjustGamma(p: number) {
  return ((p + 0.055) / 1.055) ** 2.4;
}

function relativeLuminance(o: RGBColor) {
  const rsrgb = o.r / 255;
  const gsrgb = o.g / 255;
  const bsrgb = o.b / 255;

  const r = rsrgb <= 0.03928
    ? rsrgb * lowc
    : adjustGamma(rsrgb);
  const g = gsrgb <= 0.03928
    ? gsrgb * lowc
    : adjustGamma(gsrgb);
  const b = bsrgb <= 0.03928
    ? bsrgb * lowc
    : adjustGamma(bsrgb);

  return r * rc + g * gc + b * bc;
}

export interface NodeColor {
  backgroundColor: CssColor;
  fontColor: CssColor;
}

export type NodeColorGetter = (node: HierarchyNode<Entry>) => NodeColor;

function createRainbowColor(root: HierarchyNode<Entry>): NodeColorGetter {
  const colorParentMap = new Map<number, CssColor>();
  colorParentMap.set(root.data.getID(), COLOR_BASE);
  // Use the report's complete top-level categories, including culled ones,
  // so zooming into a single category preserves its hue.
  const categories = [
    ...root.data.getChildren(),
  ].sort((a, b) => a.getSize() - b.getSize());
  const colorScale = scaleSequential([0, categories.length], n => hsl(360 * n, 0.3, 0.85));
  categories.forEach((entry, id) => colorParentMap.set(entry.getID(), colorScale(id).toString()));

  const colorMap = new Map<HierarchyNode<Entry>, NodeColor>();

  const lightScale = scaleLinear().domain([0, root.height]).range([0.9, 0.3]);

  const getBackgroundColor = (node: HierarchyNode<Entry>) => {
    const parents = node.ancestors();
    const colorStr
      = parents.length === 1
        ? colorParentMap.get(parents[0].data.getID())
        : colorParentMap.get(parents[parents.length - 2].data.getID());

    const hslColor = hsl(colorStr as string);
    hslColor.l = lightScale(node.depth);

    return hslColor;
  };

  return (node: HierarchyNode<Entry>): NodeColor => {
    if (!colorMap.has(node)) {
      const backgroundColor = getBackgroundColor(node);
      const l = relativeLuminance(backgroundColor.rgb());
      const fontColor = l > 0.19
        ? "#000"
        : "#fff";
      colorMap.set(node, {
        backgroundColor: backgroundColor.toString(),
        fontColor,
      });
    }

    return colorMap.get(node)!;
  };
}

export default createRainbowColor;
