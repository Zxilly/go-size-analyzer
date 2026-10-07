/** JSON embedded in HTML must not contain raw-text closing tags. */
export function serializeHtmlJson(value: unknown): string {
  return JSON.stringify(value).replace(/[<>&\u2028\u2029]/g, char =>
    `\\u${char.charCodeAt(0).toString(16).padStart(4, "0")}`);
}
