/**
 * Lightweight, dependency-free Markdown → HTML renderer.
 *
 * Supported
 * ─────────
 *  Blocks : headings (ATX `#` + Setext), paragraphs, fenced & indented code,
 *           blockquotes (nested, lazy, GitHub alerts), ordered/unordered/task
 *           lists (nested, tight & loose), GFM tables with alignment,
 *           thematic breaks, footnote definitions, link reference definitions.
 *  Inline : bold, italic, bold-italic, strikethrough, highlight, inline code,
 *           links, reference links, images, autolinks, bare URLs & emails,
 *           footnote references, hard breaks, backslash escapes.
 *  Math   : LaTeX math via MathJax 3 — $$…$$ display blocks, $…$ inline,
 *           \[…\] display blocks, \(…\) inline. Math is extracted before
 *           HTML-escaping so LaTeX syntax is preserved verbatim.
 *
 * Security
 * ────────
 *  The whole input is HTML-escaped up front, so raw HTML in the source is
 *  rendered as text (never executed). Every URL is passed through a scheme
 *  allow-list, so `javascript:` / `vbscript:` / non-image `data:` are dropped.
 *
 * Math rendering
 * ──────────────
 *  Math placeholders (<span data-math> / <div data-math-display>) are emitted
 *  synchronously by renderMarkdown(). The caller is responsible for running
 *  MathJax.typesetPromise() on the container element after inserting the HTML
 *  into the DOM. Use the provided `typesetMath(el)` helper, which lazy-loads
 *  MathJax 3 from CDN on first call and returns a Promise<void>.
 */

import type { Ctx, MarkdownOptions } from './types';
import { DEFAULT_CLASSES } from './styles';
import { escapeHtml } from './utils';
import { parseBlocks } from './blocks';
import { renderInline } from './inline';
import { extractDefinitions, renderFootnotes } from './definitions';
import { maskCodeRegions, extractMath, reinjectMath, type MathStash } from './math';

export type { MarkdownOptions, StyleKey } from './types';
export { typesetMath } from './math';
export { escapeHtml } from './utils';
export { renderInline } from './inline';

/* ----------------------------------------------------------------- public */

export function renderMarkdown(text: string, options: MarkdownOptions = {}): string {
  if (!text) return '';

  const ctx: Ctx = {
    opts: {
      breaks: options.breaks ?? true,
      headingIds: options.headingIds ?? true,
      linkTarget: options.linkTarget === undefined ? '_blank' : options.linkTarget,
    },
    cls: { ...DEFAULT_CLASSES, ...(options.classes ?? {}) },
    links: new Map(),
    footnoteDefs: new Map(),
    footnoteOrder: [],
    slugs: new Map(),
  };

  // 1 — strip control chars that our placeholder tokens rely on.
  const cleaned = String(text)
    .replace(/\r\n?/g, '\n')
    .replace(/[\u0000-\u0008\u000b-\u001f]/g, '');

  // 2 — extract math BEFORE HTML-escaping so LaTeX syntax is preserved
  //     verbatim. Code regions are masked only for the duration of the math
  //     extraction (so $…$ inside code stays literal) and restored right
  //     after, BEFORE escaping/parsing — otherwise code blocks would never
  //     reach the block parser and raw unescaped source would leak into the
  //     final HTML.
  let mathStash: MathStash[] = [];
  let source = cleaned;
  if (options.math ?? true) {
    const { text: codeMasked, restore } = maskCodeRegions(cleaned);
    const extracted = extractMath(codeMasked);
    mathStash = extracted.stash;
    source = restore(extracted.text);
  }

  const normalized = escapeHtml(source.replace(/\t/g, '    '));

  const lines = extractDefinitions(normalized.split('\n'), ctx);

  // 3 — link reference destinations end up inside href/title attributes; a
  //     raw math token there would corrupt the attribute, so blank them out.
  const attrSafe = (u: string) => u.replace(/\u0004M\\d+\u0004/g, '');
  ctx.links.forEach((def) => {
    def.href = attrSafe(def.href);
    if (def.title) def.title = attrSafe(def.title);
  });

  let html = parseBlocks(lines, ctx) + renderFootnotes(ctx);

  // 4 — scrub any token that landed inside an attribute (href, title, id).
  html = html.replace(/((?:href|title|id|src|alt)="[^"]*)\u0004M\\d+\u0004([^"]*")/g, '$1$2');

  // 5 — re-inject math placeholders into the final HTML.
  return reinjectMath(html, mathStash);
}

/** Render a single line/fragment without block wrappers (labels, titles, …). */
export function renderMarkdownInline(text: string, options: MarkdownOptions = {}): string {
  if (!text) return '';
  const ctx: Ctx = {
    opts: {
      breaks: options.breaks ?? false,
      headingIds: false,
      linkTarget: options.linkTarget === undefined ? '_blank' : options.linkTarget,
    },
    cls: { ...DEFAULT_CLASSES, ...(options.classes ?? {}) },
    links: new Map(),
    footnoteDefs: new Map(),
    footnoteOrder: [],
    slugs: new Map(),
  };
  return renderInline(escapeHtml(String(text).replace(/\r\n?/g, '\n')), ctx);
}

export default renderMarkdown;
