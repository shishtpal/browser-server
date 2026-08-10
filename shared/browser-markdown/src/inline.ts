import type { Ctx } from './types';
import { NUL, escapeAttr, sanitizeUrl } from './utils';

/* ------------------------------------------------------- inline rendering */

const LINK_TAIL =
  /\(\s*(?:&lt;([^&\s]*)&gt;|([^\s()]*(?:\([^\s()]*\)[^\s()]*)*))(?:\s+(?:&quot;([\s\S]*?)&quot;|&#39;([\s\S]*?)&#39;))?\s*\)/
    .source;

const INLINE_LINK_RE = new RegExp(`\\[((?:[^\\[\\]]|\\[[^\\[\\]]*\\])*)\\]${LINK_TAIL}`, 'g');
const IMAGE_RE = new RegExp(`!\\[([^\\[\\]]*)\\]${LINK_TAIL}`, 'g');

function anchor(text: string, href: string, title: string | undefined, ctx: Ctx): string {
  const safe = sanitizeUrl(href);
  const external = /^(?:https?:)?\/\//i.test(safe);
  const attrs = [`href="${safe}"`, `class="${ctx.cls.a}"`];
  if (title) attrs.push(`title="${title}"`);
  if (external && ctx.opts.linkTarget) {
    attrs.push(`target="${ctx.opts.linkTarget}"`, 'rel="noopener noreferrer nofollow"');
  }
  return `<a ${attrs.join(' ')}>${text}</a>`;
}

export function renderInline(src: string, ctx: Ctx): string {
  const stash: string[] = [];
  const hold = (html: string) => `${NUL}${stash.push(html) - 1}${NUL}`;
  let s = src;

  /* 1 — backslash escapes (also covers already-escaped entities) */
  s = s.replace(/\\(&(?:amp|lt|gt|quot|#39);|[\\`*_{}[\]()#+\-.!>~|=^$])/g, (_m, ch: string) =>
    hold(ch),
  );

  /* 2 — code spans (supports backtick runs: ``a ` b``) */
  s = s.replace(/(`+)([\s\S]*?[^`])\1(?!`)/g, (_m, _fence: string, code: string) => {
    let c = code.replace(/\n/g, ' ');
    if (c.length > 2 && c.startsWith(' ') && c.endsWith(' ') && c.trim()) c = c.slice(1, -1);
    return hold(`<code class="${ctx.cls.code}">${c}</code>`);
  });

  /* 3 — footnote references */
  s = s.replace(/\[\^([^\]\s]+)\]/g, (m, rawId: string) => {
    const id = rawId.toLowerCase();
    if (!ctx.footnoteDefs.has(id)) return m;
    let index = ctx.footnoteOrder.indexOf(id);
    if (index === -1) index = ctx.footnoteOrder.push(id) - 1;
    const key = escapeAttr(id);
    return hold(
      `<sup id="fnref-${key}"><a href="#fn-${key}" class="${ctx.cls.footnoteRef}">[${index + 1}]</a></sup>`,
    );
  });

  /* 4 — images */
  s = s.replace(
    IMAGE_RE,
    (_m, alt: string, angle: string, bare: string, t1: string, t2: string) => {
      const src2 = sanitizeUrl(angle || bare || '');
      const title = t1 || t2;
      return hold(
        `<img src="${src2}" alt="${alt}"${title ? ` title="${title}"` : ''} loading="lazy" class="${ctx.cls.img}"/>`,
      );
    },
  );

  /* 5 — inline links */
  s = s.replace(
    INLINE_LINK_RE,
    (_m, text: string, angle: string, bare: string, t1: string, t2: string) =>
      hold(anchor(renderInline(text, ctx), angle || bare || '', t1 || t2, ctx)),
  );

  /* 6 — reference links: [text][id], [text][], [id] */
  s = s.replace(
    /\[((?:[^\[\]]|\[[^\[\]]*\])*)\](?:\[([^\]]*)\])?/g,
    (m, text: string, ref?: string) => {
      const key = (ref && ref.trim() ? ref : text).trim().toLowerCase();
      const def = ctx.links.get(key);
      if (!def) return m;
      return hold(anchor(renderInline(text, ctx), def.href, def.title, ctx));
    },
  );

  /* 7 — autolinks <https://…> / <mail@example.com> */
  s = s.replace(
    /&lt;((?:https?|ftp|mailto):[^\s&]+|[^\s@&]+@[^\s@&]+\.[^\s@&]+)&gt;/g,
    (_m, url: string) => {
      const href = url.includes('@') && !url.startsWith('mailto:') ? `mailto:${url}` : url;
      return hold(anchor(url, href, undefined, ctx));
    },
  );

  /* 8 — bare URLs & emails */
  s = s.replace(
    /(^|[\s([])((?:https?:\/\/|www\.)[^\s<>"'`]+[^\s<>"'`.,;:!?)\]}])/g,
    (_m, pre: string, url: string) =>
      pre + hold(anchor(url, url.startsWith('www.') ? `http://${url}` : url, undefined, ctx)),
  );
  s = s.replace(
    /(^|[\s([])([\w.+-]+@[\w-]+(?:\.[\w-]+)+)/g,
    (_m, pre: string, mail: string) => pre + hold(anchor(mail, `mailto:${mail}`, undefined, ctx)),
  );

  /* 9 — emphasis (strongest first) */
  const { strong, em, del, mark } = ctx.cls;
  s = s
    .replace(
      /\*\*\*([^\s*](?:[\s\S]*?[^\s*])?)\*\*\*/g,
      `<strong class="${strong}"><em class="${em}">$1</em></strong>`,
    )
    .replace(
      /(^|[^\w\\])___([^\s_](?:[\s\S]*?[^\s_])?)___(?!\w)/g,
      `$1<strong class="${strong}"><em class="${em}">$2</em></strong>`,
    )
    .replace(/\*\*([^\s*](?:[\s\S]*?[^\s*])?)\*\*/g, `<strong class="${strong}">$1</strong>`)
    .replace(
      /(^|[^\w\\])__([^\s_](?:[\s\S]*?[^\s_])?)__(?!\w)/g,
      `$1<strong class="${strong}">$2</strong>`,
    )
    .replace(/\*([^\s*](?:[\s\S]*?[^\s*])?)\*/g, `<em class="${em}">$1</em>`)
    .replace(/(^|[^\w\\])_([^\s_](?:[\s\S]*?[^\s_])?)_(?!\w)/g, `$1<em class="${em}">$2</em>`)
    .replace(/~~([^\s~](?:[\s\S]*?[^\s~])?)~~/g, `<del class="${del}">$1</del>`)
    .replace(/==([^\s=](?:[\s\S]*?[^\s=])?)==/g, `<mark class="${mark}">$1</mark>`);

  /* 10 — line breaks */
  s = s.replace(/(?: {2,}|\\)\n/g, '<br/>\n');
  s = ctx.opts.breaks ? s.replace(/\n(?!$)/g, '<br/>\n') : s;

  /* 11 — restore stashed fragments (nested placeholders need a few passes) */
  const token = new RegExp(`${NUL}(\\d+)${NUL}`, 'g');
  for (let pass = 0; pass < 6 && s.includes(NUL); pass++) {
    s = s.replace(token, (_m, n: string) => stash[Number(n)] ?? '');
  }
  return s;
}
