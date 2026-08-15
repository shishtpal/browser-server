import type { Ctx } from './types';
import {
  ATX_RE,
  FENCE_RE,
  HR_RE,
  INDENT_CODE_RE,
  LIST_RE,
  TABLE_DELIM_RE,
  isBlank,
  isBlockStart,
  stripBlockquoteMarker,
} from './patterns';
import { renderInline } from './inline';
import { alignStyle, codeBlockHtml, headingHtml, splitTableRow } from './block-renderers';
import { ALERTS } from './styles';

/* ------------------------------------------------------------ block parse */

export function parseBlocks(lines: string[], ctx: Ctx, tight = false): string {
  const out: string[] = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    if (isBlank(line)) {
      i++;
      continue;
    }

    /* ── fenced code ─────────────────────────────────────────────── */
    const fence = FENCE_RE.exec(line);
    if (fence && !(fence[2][0] === '`' && fence[3].includes('`'))) {
      const [, indent, marker, info] = fence;
      const close = new RegExp(`^ {0,3}\\${marker[0]}{${marker.length},}[ \\t]*$`);
      const body: string[] = [];
      i++;
      while (i < lines.length && !close.test(lines[i])) {
        body.push(lines[i].startsWith(indent) ? lines[i].slice(indent.length) : lines[i]);
        i++;
      }
      i++; // consume closing fence
      out.push(codeBlockHtml(body.join('\n').replace(/\n+$/, ''), info, ctx));
      continue;
    }

    /* ── indented code ───────────────────────────────────────────── */
    if (INDENT_CODE_RE.test(line)) {
      const body: string[] = [];
      while (i < lines.length) {
        if (INDENT_CODE_RE.test(lines[i])) {
          body.push(lines[i].slice(4));
          i++;
          continue;
        }
        if (isBlank(lines[i]) && i + 1 < lines.length && INDENT_CODE_RE.test(lines[i + 1])) {
          body.push('');
          i++;
          continue;
        }
        break;
      }
      out.push(codeBlockHtml(body.join('\n'), '', ctx));
      continue;
    }

    /* ── thematic break ──────────────────────────────────────────── */
    if (HR_RE.test(line)) {
      out.push(`<hr class="${ctx.cls.hr}"/>`);
      i++;
      continue;
    }

    /* ── ATX heading ─────────────────────────────────────────────── */
    const atx = ATX_RE.exec(line);
    if (atx) {
      out.push(headingHtml(atx[1].length, atx[2] ?? '', ctx));
      i++;
      continue;
    }

    /* ── blockquote (with GitHub alerts) ─────────────────────────── */
    if (stripBlockquoteMarker(line) !== null) {
      const body: string[] = [];
      while (i < lines.length) {
        const stripped = stripBlockquoteMarker(lines[i]);
        if (stripped !== null) {
          body.push(stripped);
          i++;
          continue;
        }
        if (!isBlank(lines[i]) && !isBlockStart(lines, i)) {
          body.push(lines[i]);
          i++;
          continue;
        } // lazy
        break;
      }
      const alert = /^\[!(note|tip|important|warning|caution)\][ \t]*$/i.exec(
        body[0]?.trim() ?? '',
      );
      if (alert) {
        const meta = ALERTS[alert[1].toLowerCase()];
        const inner = parseBlocks(body.slice(1), ctx);
        out.push(
          `<div class="${ctx.cls.alert} ${meta.cls}">` +
            `<p class="${ctx.cls.alertTitle}"><span aria-hidden="true">${meta.icon}</span>${meta.label}</p>${inner}</div>`,
        );
      } else {
        out.push(
          `<blockquote class="${ctx.cls.blockquote}">${parseBlocks(body, ctx)}</blockquote>`,
        );
      }
      continue;
    }

    /* ── table ───────────────────────────────────────────────────── */
    if (line.includes('|') && i + 1 < lines.length && TABLE_DELIM_RE.test(lines[i + 1])) {
      const header = splitTableRow(line);
      const aligns = splitTableRow(lines[i + 1]).map((c) => {
        const left = c.startsWith(':');
        const right = c.endsWith(':');
        return left && right ? 'center' : right ? 'right' : left ? 'left' : null;
      });
      if (header.length === aligns.length) {
        i += 2;
        const rows: string[][] = [];
        while (i < lines.length && !isBlank(lines[i]) && lines[i].includes('|')) {
          rows.push(splitTableRow(lines[i]));
          i++;
        }
        const head =
          `<tr class="${ctx.cls.tr}">` +
          header
            .map(
              (c, k) =>
                `<th class="${ctx.cls.th}"${alignStyle(aligns[k])} scope="col">${renderInline(c, ctx)}</th>`,
            )
            .join('') +
          '</tr>';
        const bodyHtml = rows
          .map((cells) => {
            const padded = Array.from({ length: header.length }, (_, k) => cells[k] ?? '');
            return (
              `<tr class="${ctx.cls.tr}">` +
              padded
                .map(
                  (c, k) =>
                    `<td class="${ctx.cls.td}"${alignStyle(aligns[k])}>${renderInline(c, ctx)}</td>`,
                )
                .join('') +
              '</tr>'
            );
          })
          .join('');
        out.push(
          `<div class="${ctx.cls.tableWrap}"><table class="${ctx.cls.table}">` +
            `<thead class="${ctx.cls.thead}">${head}</thead>` +
            `<tbody class="${ctx.cls.tbody}">${bodyHtml}</tbody></table></div>`,
        );
        continue;
      }
    }

    /* ── lists ───────────────────────────────────────────────────── */
    if (LIST_RE.test(line)) {
      const { html, next } = parseList(lines, i, ctx);
      out.push(html);
      i = next;
      continue;
    }

    /* ── paragraph / setext heading ──────────────────────────────── */
    const buf: string[] = [];
    let setext: 1 | 2 | null = null;
    while (i < lines.length) {
      const cur = lines[i];
      if (isBlank(cur)) break;
      if (buf.length) {
        const s = /^ {0,3}(=+|-+)[ \t]*$/.exec(cur);
        if (s) {
          setext = s[1][0] === '=' ? 1 : 2;
          i++;
          break;
        }
        if (isBlockStart(lines, i)) break;
      }
      buf.push(cur.replace(/^ {0,3}/, ''));
      i++;
    }
    if (!buf.length) {
      i++;
      continue;
    }
    if (setext) {
      out.push(headingHtml(setext, buf.join(' '), ctx));
      continue;
    }

    const inner = renderInline(buf.join('\n'), ctx);
    out.push(tight ? inner : `<p class="${ctx.cls.p}">${inner}</p>`);
  }

  return out.join('\n');
}

/* --------------------------------------------------------------- list ADT */

function parseList(lines: string[], start: number, ctx: Ctx): { html: string; next: number } {
  const first = LIST_RE.exec(lines[start])!;
  const ordered = /\d/.test(first[2]);
  const startNumber = ordered ? parseInt(first[2], 10) : 1;
  const baseIndent = first[1].length;

  const items: string[][] = [];
  let loose = false;
  let pendingBlank = false;
  let contentIndent = 0;
  let i = start;

  while (i < lines.length) {
    const line = lines[i];

    if (isBlank(line)) {
      pendingBlank = true;
      i++;
      continue;
    }

    const indent = line.length - line.replace(/^ +/, '').length;
    const item = LIST_RE.exec(line);

    // New sibling item at (roughly) the same level.
    if (item && item[1].length <= baseIndent + 1) {
      if (/\d/.test(item[2]) !== ordered) break; // different list type → new list
      if (pendingBlank && items.length) loose = true;
      pendingBlank = false;
      contentIndent = item[1].length + item[2].length + Math.min((item[3] ?? ' ').length, 4);
      items.push([item[4] ?? '']);
      i++;
      continue;
    }

    if (!items.length) break;

    // Continuation belonging to the current item.
    if (indent >= contentIndent) {
      if (pendingBlank) {
        items[items.length - 1].push('');
        loose = true;
        pendingBlank = false;
      }
      items[items.length - 1].push(line.slice(contentIndent));
      i++;
      continue;
    }

    // Lazy paragraph continuation.
    if (!pendingBlank && !isBlockStart(lines, i)) {
      items[items.length - 1].push(line.trim());
      i++;
      continue;
    }

    break;
  }

  let hasTask = false;
  const rendered = items.map((item) => {
    const raw = item.join('\n');
    const task = /^\[([ xX])\](?:[ \t]+|$)([\s\S]*)$/.exec(raw);
    if (task) {
      hasTask = true;
      const checked = task[1].toLowerCase() === 'x';
      const inner = parseBlocks(task[2].split('\n'), ctx, !loose);
      return (
        `<li class="${ctx.cls.taskItem}">` +
        `<input type="checkbox" disabled${checked ? ' checked' : ''} class="${ctx.cls.checkbox}"/>` +
        `<span${checked ? ' class="opacity-70"' : ''}>${inner}</span></li>`
      );
    }
    return `<li class="${ctx.cls.li}">${parseBlocks(item, ctx, !loose)}</li>`;
  });

  const tag = ordered ? 'ol' : 'ul';
  const cls = hasTask ? ctx.cls.taskList : ctx.cls[tag];
  const startAttr = ordered && startNumber !== 1 ? ` start="${startNumber}"` : '';
  return { html: `<${tag} class="${cls}"${startAttr}>${rendered.join('')}</${tag}>`, next: i };
}
