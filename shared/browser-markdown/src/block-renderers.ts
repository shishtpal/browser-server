import type { Ctx, StyleKey } from './types';
import { renderInline } from './inline';
import { escapeAttr, slugify } from './utils';

/* -------------------------------------------------------- block renderers */

export function codeBlockHtml(code: string, info: string, ctx: Ctx): string {
  const lang =
    info
      .trim()
      .split(/\s+/)[0]
      ?.replace(/[^\w+#.-]/g, '')
      .slice(0, 24)
      .toLowerCase() ?? '';
  const label = lang ? `<span class="${ctx.cls.codeLang}">${lang}</span>` : '';
  return (
    `<div class="${ctx.cls.codeWrap}" data-lang="${lang}">${label}` +
    `<button type="button" data-copy-code class="${ctx.cls.copyButton}" title="Copy code" aria-label="Copy code">Copy</button>` +
    `<pre class="${ctx.cls.pre}"><code class="language-${lang || 'plaintext'}">${code}</code></pre></div>`
  );
}

export function headingHtml(level: number, raw: string, ctx: Ctx): string {
  const inner = renderInline(raw.trim(), ctx);
  const key = `h${level}` as StyleKey;
  const id = ctx.opts.headingIds ? ` id="${escapeAttr(slugify(inner, ctx.slugs))}"` : '';
  return `<h${level}${id} class="${ctx.cls[key]}">${inner}</h${level}>`;
}

export function splitTableRow(row: string): string[] {
  const cells: string[] = [];
  let cur = '';
  for (let i = 0; i < row.length; i++) {
    const ch = row[i];
    if (ch === '\\' && row[i + 1] === '|') {
      cur += '|';
      i++;
      continue;
    }
    if (ch === '|') {
      cells.push(cur);
      cur = '';
      continue;
    }
    cur += ch;
  }
  cells.push(cur);
  const trimmed = row.trim();
  if (trimmed.startsWith('|')) cells.shift();
  if (trimmed.length > 1 && trimmed.endsWith('|') && !trimmed.endsWith('\\|')) cells.pop();
  return cells.map((c) => c.trim());
}

export function alignStyle(align: string | null): string {
  return align ? ` style="text-align:${align}"` : '';
}
