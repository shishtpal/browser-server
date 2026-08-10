import type { Ctx } from './types';
import { FENCE_RE, INDENT_CODE_RE, isBlank } from './patterns';
import { parseBlocks } from './blocks';
import { escapeAttr } from './utils';

/* ---------------------------------------------------------- pre-pass defs */

const FOOTNOTE_DEF_RE = /^ {0,3}\[\^([^\]\s]+)\]:[ \t]*(.*)$/;
const LINK_DEF_RE =
  /^ {0,3}\[([^\]^][^\]]*)\]:[ \t]*(?:&lt;([^&\s]*)&gt;|(\S+))(?:[ \t]+(?:&quot;([\s\S]*?)&quot;|&#39;([\s\S]*?)&#39;|\(([^)]*)\)))?[ \t]*$/;

/** Pulls link-reference and footnote definitions out of the document. */
export function extractDefinitions(lines: string[], ctx: Ctx): string[] {
  const out: string[] = [];
  let fence: string | null = null;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    const f = FENCE_RE.exec(line);
    if (f) {
      if (fence && new RegExp(`^ {0,3}\\${fence[0]}{${fence.length},}[ \t]*$`).test(line))
        fence = null;
      else if (!fence) fence = f[2];
      out.push(line);
      continue;
    }
    if (fence) {
      out.push(line);
      continue;
    }

    const fn = FOOTNOTE_DEF_RE.exec(line);
    if (fn) {
      const body = [fn[2]];
      while (i + 1 < lines.length) {
        const next = lines[i + 1];
        if (INDENT_CODE_RE.test(next) || /^ {4}/.test(next)) {
          body.push(next.slice(4));
          i++;
          continue;
        }
        if (isBlank(next) && /^ {4}\S/.test(lines[i + 2] ?? '')) {
          body.push('');
          i++;
          continue;
        }
        break;
      }
      ctx.footnoteDefs.set(fn[1].toLowerCase(), body);
      continue;
    }

    const def = LINK_DEF_RE.exec(line);
    if (def) {
      ctx.links.set(def[1].trim().toLowerCase(), {
        href: def[2] || def[3] || '',
        title: def[4] || def[5] || def[6] || undefined,
      });
      continue;
    }

    out.push(line);
  }
  return out;
}

export function renderFootnotes(ctx: Ctx): string {
  if (!ctx.footnoteOrder.length) return '';
  const items = ctx.footnoteOrder
    .map((id) => {
      const key = escapeAttr(id);
      const inner = parseBlocks(ctx.footnoteDefs.get(id) ?? [''], ctx, true);
      return (
        `<li id="fn-${key}" class="${ctx.cls.li}">${inner}` +
        `<a href="#fnref-${key}" class="${ctx.cls.footnoteBackref}" aria-label="Back to reference">↩</a></li>`
      );
    })
    .join('');
  return (
    `<section class="${ctx.cls.footnotes}" role="doc-endnotes">` +
    `<hr class="${ctx.cls.hr}"/><ol class="${ctx.cls.ol}">${items}</ol></section>`
  );
}
