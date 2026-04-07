import { layoutWithLines, prepareWithSegments } from "@chenglou/pretext";
import { PADDING, TOP_PADDING } from "./const.ts";

const FONT_SIZE_PX = 12.8; // 0.8em * 16px base
const FONT_STRING = `${FONT_SIZE_PX}px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif`;

const measureCache = new Map<string, [number, number]>();

export function measureText(text: string): [number, number] {
  const cached = measureCache.get(text);
  if (cached) {
    return cached;
  }

  const prepared = prepareWithSegments(text, FONT_STRING);
  const result = layoutWithLines(prepared, Infinity, FONT_SIZE_PX);

  const width = result.lines.length > 0 ? result.lines[0].width : 0;
  const entry: [number, number] = [width, result.height];
  measureCache.set(text, entry);
  return entry;
}

const splitter = /[/\\]/;
const verMatcher = /v\d+/;

export function getShortName(title: string): string {
  const words = title.split(splitter);
  const last = words[words.length - 1];

  if (words.length >= 2 && verMatcher.test(last)) {
    const split = title[title.length - last.length - 1];
    return `${words[words.length - 2]}${split}${last}`;
  }

  return words[words.length - 1];
}

function getScaleInternal(
  title: string,
  width: number,
  height: number,
  hasChildren: boolean,
  fallback: boolean,
): [string, number] {
  if (title === "") {
    return ["", 0];
  }

  const [textWidth, textHeight] = measureText(title);

  let scale: number;
  if (hasChildren) {
    scale = Math.min(
      (width * 0.9) / textWidth,
      Math.min(height, TOP_PADDING + PADDING) / textHeight,
    );
    scale = Math.min(1, scale);
  }
  else {
    scale = Math.min(
      (width * 0.9) / textWidth,
      (height * 0.9) / textHeight,
    );
    if (scale > 1) {
      scale = Math.sqrt(scale);
    }
    if (scale === Infinity) {
      scale = 1;
    }
  }

  if (scale < 0.7 && fallback) {
    return getScaleInternal(getShortName(title), width, height, hasChildren, false);
  }
  return [title, scale];
}

export function getScale(
  title: string,
  width: number,
  height: number,
  hasChildren: boolean,
): [string, number] {
  return getScaleInternal(title, width, height, hasChildren, true);
}
