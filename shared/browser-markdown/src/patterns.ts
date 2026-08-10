/* -------------------------------------------------------------- structure */

export const HR_RE = /^ {0,3}(?:(?:\*[ \t]*){3,}|(?:-[ \t]*){3,}|(?:_[ \t]*){3,})$/;
export const ATX_RE = /^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*(?:[ \t]#+)?[ \t]*$/;
export const FENCE_RE = /^( {0,3})(`{3,}|~{3,})[ \t]*(.*)$/;
export const LIST_RE = /^( {0,7})([-+*]|\d{1,9}[.)])(?:([ \t]+)(.*)|)$/;
export const TABLE_DELIM_RE = /^ {0,3}\|?[ \t]*:?-+:?[ \t]*(?:\|[ \t]*:?-+:?[ \t]*)*\|?[ \t]*$/;
export const INDENT_CODE_RE = /^ {4}(?=\S)/;

export function isBlank(line: string): boolean {
  return !line.trim();
}

export function isBlockStart(lines: string[], i: number): boolean {
  const line = lines[i];
  return (
    FENCE_RE.test(line) ||
    ATX_RE.test(line) ||
    HR_RE.test(line) ||
    /^ {0,3}>/.test(line) ||
    /^ {0,3}(?:[-+*]|\d{1,9}[.)])(?:[ \t]|$)/.test(line) ||
    (line.includes('|') && i + 1 < lines.length && TABLE_DELIM_RE.test(lines[i + 1]))
  );
}
