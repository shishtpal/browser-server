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

/* ------------------------------------------------------------------ types */

export type StyleKey =
  | 'h1'
  | 'h2'
  | 'h3'
  | 'h4'
  | 'h5'
  | 'h6'
  | 'p'
  | 'a'
  | 'strong'
  | 'em'
  | 'del'
  | 'mark'
  | 'ul'
  | 'ol'
  | 'li'
  | 'taskList'
  | 'taskItem'
  | 'checkbox'
  | 'blockquote'
  | 'alert'
  | 'alertTitle'
  | 'hr'
  | 'img'
  | 'tableWrap'
  | 'table'
  | 'thead'
  | 'tbody'
  | 'tr'
  | 'th'
  | 'td'
  | 'code'
  | 'pre'
  | 'codeWrap'
  | 'codeLang'
  | 'copyButton'
  | 'footnotes'
  | 'footnoteRef'
  | 'footnoteBackref';

export interface MarkdownOptions {
  /** Render single newlines as `<br/>` (chat-style). Default: `true`. */
  breaks?: boolean;
  /** Add `id="…"` slugs to headings. Default: `true`. */
  headingIds?: boolean;
  /** `target` for external links, e.g. `'_blank'`. Default: `'_blank'`. */
  linkTarget?: string | null;
  /** Override any of the default Tailwind class strings. */
  classes?: Partial<Record<StyleKey, string>>;
}

/* ---------------------------------------------------------------- styling */

const DEFAULT_CLASSES: Record<StyleKey, string> = {
  h1: 'mt-5 mb-2 text-lg font-black tracking-tight',
  h2: 'mt-4 mb-2 text-base font-bold',
  h3: 'mt-4 mb-1 font-bold',
  h4: 'mt-3 mb-1 font-bold',
  h5: 'mt-3 mb-1 text-[0.95em] font-semibold',
  h6: 'mt-3 mb-1 text-[0.9em] font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400',
  p: 'my-2 leading-relaxed',
  a: 'font-medium text-sky-600 underline underline-offset-2 hover:text-sky-500 dark:text-sky-400',
  strong: 'font-semibold',
  em: 'italic',
  del: 'line-through opacity-70',
  mark: 'rounded bg-yellow-200 px-1 dark:bg-yellow-500/30',
  ul: 'my-2 ml-5 list-disc space-y-1',
  ol: 'my-2 ml-5 list-decimal space-y-1',
  li: 'pl-1 marker:text-slate-400',
  taskList: 'my-2 ml-1 list-none space-y-1',
  taskItem: 'flex items-start gap-2',
  checkbox: 'mt-1 h-3.5 w-3.5 shrink-0 accent-sky-600',
  blockquote:
    'my-3 border-l-4 border-slate-300 pl-4 text-slate-600 italic dark:border-slate-600 dark:text-slate-300',
  alert: 'my-3 rounded-lg border-l-4 px-4 py-2 text-[0.95em] not-italic',
  alertTitle: 'mb-1 flex items-center gap-1.5 font-semibold',
  hr: 'my-4 border-t border-slate-200 dark:border-slate-700',
  img: 'my-3 max-w-full rounded-lg',
  tableWrap:
    'not-prose my-3 overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-700',
  table: 'w-full border-collapse text-left text-[0.9em]',
  thead: 'bg-slate-50 dark:bg-slate-800',
  tbody: 'divide-y divide-slate-200 dark:divide-slate-700',
  tr: '',
  th: 'px-3 py-2 font-semibold whitespace-nowrap',
  td: 'px-3 py-2 align-top',
  code: 'rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[0.85em] dark:bg-slate-800',
  pre: 'overflow-x-auto rounded-lg bg-slate-800 p-3 pr-14 text-[0.85em] text-slate-50 dark:bg-slate-900 dark:text-slate-100',
  codeWrap: 'not-prose group/code relative my-3',
  codeLang:
    'absolute left-3 top-2 select-none text-[0.65em] font-semibold uppercase tracking-wider text-slate-400',
  copyButton:
    'absolute right-2 top-2 rounded-md bg-slate-200/90 px-2 py-1 text-[0.7em] font-semibold text-slate-600 transition hover:bg-slate-300 sm:opacity-0 sm:group-hover/code:opacity-100 sm:focus:opacity-100 dark:bg-slate-700/90 dark:text-slate-200 dark:hover:bg-slate-600',
  footnotes: 'mt-6 text-[0.85em] text-slate-600 dark:text-slate-400',
  footnoteRef: 'text-sky-600 no-underline dark:text-sky-400',
  footnoteBackref: 'ml-1 text-sky-600 no-underline dark:text-sky-400',
};

const ALERTS: Record<string, { label: string; icon: string; cls: string }> = {
  note: {
    label: 'Note',
    icon: 'ℹ️',
    cls: 'border-sky-500 bg-sky-50 text-sky-900 dark:bg-sky-500/10 dark:text-sky-200',
  },
  tip: {
    label: 'Tip',
    icon: '💡',
    cls: 'border-emerald-500 bg-emerald-50 text-emerald-900 dark:bg-emerald-500/10 dark:text-emerald-200',
  },
  important: {
    label: 'Important',
    icon: '❗',
    cls: 'border-violet-500 bg-violet-50 text-violet-900 dark:bg-violet-500/10 dark:text-violet-200',
  },
  warning: {
    label: 'Warning',
    icon: '⚠️',
    cls: 'border-amber-500 bg-amber-50 text-amber-900 dark:bg-amber-500/10 dark:text-amber-200',
  },
  caution: {
    label: 'Caution',
    icon: '🛑',
    cls: 'border-rose-500 bg-rose-50 text-rose-900 dark:bg-rose-500/10 dark:text-rose-200',
  },
};

/* ------------------------------------------------------------------ utils */

const NUL = '\u0000';

const HTML_ESCAPES: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
};

export function escapeHtml(str: string): string {
  return str.replace(/[&<>"']/g, (c) => HTML_ESCAPES[c]);
}

/** Keep only attribute-safe characters (used for ids). */
function escapeAttr(str: string): string {
  return str.replace(/[^\w.:-]/g, '-');
}

const SAFE_SCHEMES = /^(?:https?|mailto|tel|sms|ftp):/i;

/** Input is already HTML-escaped; we only need to police the scheme. */
function sanitizeUrl(raw: string): string {
  const url = raw.trim().replace(/^&lt;/, '').replace(/&gt;$/, '');
  const probe = url.replace(/[\s\u0000-\u001f]/g, '').toLowerCase();
  if (/^data:image\/(?:png|jpe?g|gif|webp|avif|svg\+xml);base64,/.test(probe)) return url;
  if (/^[a-z][a-z0-9+.-]*:/.test(probe) && !SAFE_SCHEMES.test(probe)) return '#';
  return url;
}

function slugify(text: string, taken: Map<string, number>): string {
  const base =
    text
      .replace(/<[^>]*>/g, '')
      .replace(/&[a-z0-9#]+;/gi, '')
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, '')
      .trim()
      .replace(/\s+/g, '-')
      .slice(0, 64) || 'section';
  const seen = taken.get(base) ?? 0;
  taken.set(base, seen + 1);
  return seen ? `${base}-${seen}` : base;
}

/* -------------------------------------------------------------- structure */

interface Ctx {
  opts: Required<Omit<MarkdownOptions, 'classes'>>;
  cls: Record<StyleKey, string>;
  links: Map<string, { href: string; title?: string }>;
  footnoteDefs: Map<string, string[]>;
  footnoteOrder: string[];
  slugs: Map<string, number>;
}

const HR_RE = /^ {0,3}(?:(?:\*[ \t]*){3,}|(?:-[ \t]*){3,}|(?:_[ \t]*){3,})$/;
const ATX_RE = /^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*(?:[ \t]#+)?[ \t]*$/;
const FENCE_RE = /^( {0,3})(`{3,}|~{3,})[ \t]*(.*)$/;
const LIST_RE = /^( {0,7})([-+*]|\d{1,9}[.)])(?:([ \t]+)(.*)|)$/;
const TABLE_DELIM_RE = /^ {0,3}\|?[ \t]*:?-+:?[ \t]*(?:\|[ \t]*:?-+:?[ \t]*)*\|?[ \t]*$/;
const INDENT_CODE_RE = /^ {4}(?=\S)/;

function isBlank(line: string): boolean {
  return !line.trim();
}

function isBlockStart(lines: string[], i: number): boolean {
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

/* -------------------------------------------------------- block renderers */

function codeBlockHtml(code: string, info: string, ctx: Ctx): string {
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

function headingHtml(level: number, raw: string, ctx: Ctx): string {
  const inner = renderInline(raw.trim(), ctx);
  const key = `h${level}` as StyleKey;
  const id = ctx.opts.headingIds ? ` id="${escapeAttr(slugify(inner, ctx.slugs))}"` : '';
  return `<h${level}${id} class="${ctx.cls[key]}">${inner}</h${level}>`;
}

function splitTableRow(row: string): string[] {
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

function alignStyle(align: string | null): string {
  return align ? ` style="text-align:${align}"` : '';
}

/* ------------------------------------------------------------ block parse */

function parseBlocks(lines: string[], ctx: Ctx, tight = false): string {
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
    if (/^ {0,3}>/.test(line)) {
      const body: string[] = [];
      while (i < lines.length) {
        if (/^ {0,3}>/.test(lines[i])) {
          body.push(lines[i].replace(/^ {0,3}>[ \t]?/, ''));
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

/* ---------------------------------------------------------- pre-pass defs */

const FOOTNOTE_DEF_RE = /^ {0,3}\[\^([^\]\s]+)\]:[ \t]*(.*)$/;
const LINK_DEF_RE =
  /^ {0,3}\[([^\]^][^\]]*)\]:[ \t]*(?:&lt;([^&\s]*)&gt;|(\S+))(?:[ \t]+(?:&quot;([\s\S]*?)&quot;|&#39;([\s\S]*?)&#39;|\(([^)]*)\)))?[ \t]*$/;

/** Pulls link-reference and footnote definitions out of the document. */
function extractDefinitions(lines: string[], ctx: Ctx): string[] {
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

function renderFootnotes(ctx: Ctx): string {
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

/* --------- math helpers --------- */

/**
 * MathJax 3 global interface (minimal surface we need).
 * MathJax is loaded lazily from CDN on first call to typesetMath().
 */
declare global {
  interface Window {
    MathJax?: {
      typesetPromise(nodes?: Element[]): Promise<void>;
      startup?: { promise?: Promise<unknown>; [key: string]: unknown };
      [key: string]: unknown;
    };
  }
}

const MATHJAX_CDN = 'https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-svg.js';

let _mathJaxLoading: Promise<void> | null = null;

/** Lazily load MathJax 3 from CDN (called once per page). */
function loadMathJax(): Promise<void> {
  if (_mathJaxLoading) return _mathJaxLoading;

  _mathJaxLoading = new Promise<void>((resolve, reject) => {
    if (window.MathJax?.typesetPromise) {
      resolve();
      return;
    }

    // Configure MathJax before the script loads.
    // This is a config-only object; MathJax replaces it with the real
    // instance after loading, so we bypass the type here intentionally.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).MathJax = {
      tex: {
        inlineMath: [
          ['$', '$'],
          ['\\(', '\\)'],
        ],
        displayMath: [
          ['$$', '$$'],
          ['\\[', '\\]'],
        ],
        processEscapes: true,
      },
      svg: { fontCache: 'global' },
      startup: { typeset: false }, // we call typesetPromise manually
    };

    const script = document.createElement('script');
    script.src = MATHJAX_CDN;
    script.async = true;
    script.onload = () =>
      // MathJax signals readiness via startup.promise
      (window.MathJax!.startup!.promise as Promise<unknown>).then(() => resolve()).catch(reject);
    script.onerror = () => reject(new Error('Failed to load MathJax'));
    document.head.appendChild(script);
  });

  // Allow a retry on the next call if loading failed (network hiccup etc.).
  _mathJaxLoading.catch(() => {
    _mathJaxLoading = null;
  });

  return _mathJaxLoading;
}

/**
 * Typeset all math placeholders inside `el`.
 * Safe to call before MathJax is loaded — will load it on demand.
 * Returns a Promise that resolves when typesetting is complete.
 */
export async function typesetMath(el: Element): Promise<void> {
  const placeholders = Array.from(
    el.querySelectorAll('[data-math],[data-math-display]'),
  ) as Element[];
  if (!placeholders.length) return;

  // Show the raw $…$ source immediately so content is never a blank gap if
  // the CDN is slow/unreachable; MathJax replaces it once typeset.
  for (const node of placeholders) {
    const raw = node.getAttribute('data-math') ?? node.getAttribute('data-math-display') ?? '';
    const isDisplay = node.hasAttribute('data-math-display');
    const delim = isDisplay ? ['$$', '$$'] : ['$', '$'];
    node.textContent = `${delim[0]}${decodeURIComponent(raw)}${delim[1]}`;
    node.removeAttribute('data-math');
    node.removeAttribute('data-math-display');
  }

  try {
    await loadMathJax();
    await window.MathJax!.typesetPromise(placeholders);
  } catch {
    // Leave the raw LaTeX source visible rather than a blank gap or an
    // unhandled rejection; the next render will retry.
  }
}

/* ------------------------------------------------ math pre-pass (pre-HTML) */

interface MathStash {
  token: string;
  isDisplay: boolean;
  latex: string;
}

/* ------------------------------------------------ math pre-pass (pre-HTML) */

/** Placeholder tokens use \u0004 … \u0004 (EOT), which never appears in real
 *  input (we strip it up front along with \u0000), so user text can never
 *  collide with a generated token. */
const MATH_TOKEN_PREFIX = '\u0004M';
const MATH_TOKEN_SUFFIX = '\u0004';
const CODE_TOKEN_PREFIX = '\u0004C';

/**
 * Mask regions where math must NOT be extracted (code spans, fenced code
 * blocks, indented code) so $…$ inside code stays literal, and decode them
 * back afterwards. Tokens survive HTML-escaping untouched.
 */
function maskCodeRegions(text: string): { text: string; restore: (s: string) => string } {
  const stash: string[] = [];
  const hold = (frag: string) => `${CODE_TOKEN_PREFIX}${stash.push(frag) - 1}${MATH_TOKEN_SUFFIX}`;

  let s = text;

  // 0 — link/image destinations: the (url "title") part is an attribute
  //     context; $…$ there must never become math. (Link text stays
  //     unmasked so math inside it still renders.)
  s = s.replace(/(?<=\])\([^()\n]*\)/g, (m) => hold(m));

  // 1 — fenced code blocks (``` / ~~~ …), must run before spans so multi-line
  //     fences win over any backticks inside them.
  s = s.replace(/^( {0,3})(`{3,}|~{3,})[^\n]*\n[\s\S]*?(?:^ {0,3}\2[ \t]*$|\n?(?![\s\S]))/gm, (m) =>
    hold(m),
  );

  // 2 — indented code blocks (4+ spaces, at line start)
  s = s.replace(/^(?: {4}[^\n]*(?:\n|$))+/gm, (m) => hold(m.replace(/\n+$/, '\n')));

  // 3 — inline code spans (backtick runs: ``a ` b``)
  s = s.replace(/(`+)([\s\S]*?)\1(?!`)/g, (m) => hold(m));

  const restore = (out: string) => {
    const re = new RegExp(`${CODE_TOKEN_PREFIX}(\\d+)${MATH_TOKEN_SUFFIX}`, 'g');
    for (let pass = 0; pass < 4 && out.includes(CODE_TOKEN_PREFIX); pass++) {
      out = out.replace(re, (_m, i: string) => stash[Number(i)] ?? '');
    }
    return out;
  };

  return { text: s, restore };
}

/**
 * Extract math expressions from raw (un-escaped) Markdown text BEFORE HTML
 * escaping, replacing them with unique tokens.  The stash is used later by
 * `reinjectMath` to emit safe placeholder HTML.
 *
 * Supported delimiters (in priority order):
 *   $$…$$  — display (may span multiple lines)
 *   \[…\]  — display (may span multiple lines)
 *   $…$    — inline  (single line, not empty, not starting/ending with space)
 *   \(…\)  — inline
 */
function extractMath(text: string): { text: string; stash: MathStash[] } {
  const stash: MathStash[] = [];
  let counter = 0;

  const token = (isDisplay: boolean, latex: string): string => {
    const id = `${MATH_TOKEN_PREFIX}${counter++}${MATH_TOKEN_SUFFIX}`;
    stash.push({ token: id, isDisplay, latex });
    return id;
  };

  let s = text;

  // Display: $$…$$  (skip $$ escaped with a backslash)
  s = s.replace(
    /(^|[^\\])\$\$([\s\S]*?)\$\$/g,
    (_m, pre: string, inner: string) => pre + token(true, inner),
  );

  // Display: \[…\]  (skip \\[ escaped with an extra backslash)
  s = s.replace(
    /(^|[^\\])\\\[([\s\S]*?)\\\]/g,
    (_m, pre: string, inner: string) => pre + token(true, inner),
  );

  // Inline: \(…\)  (same escape rule)
  s = s.replace(
    /(^|[^\\])\\\(([\s\S]*?)\\\)/g,
    (_m, pre: string, inner: string) => pre + token(false, inner),
  );

  // Inline: $…$  (single line, non-empty, no leading/trailing space,
  // not preceded/followed by a digit, and not escaped). This prevents
  // currency text like "$5 and $10" from being treated as math.
  s = s.replace(
    /(^|[^\\$\d])\$(?!\$)([^\n$][^$\n]*?[^\s$]|[^\s$])\$(?!\$|\d)/g,
    (_m, pre: string, inner: string) => pre + token(false, inner),
  );

  return { text: s, stash };
}

/**
 * After HTML-escaping, replace math tokens with placeholder elements that
 * carry the LaTeX in a `data-math` / `data-math-display` attribute.
 * `typesetMath()` decodes these and hands them to MathJax.
 */
function reinjectMath(html: string, stash: MathStash[]): string {
  for (const { token, isDisplay, latex } of stash) {
    const encoded = encodeURIComponent(latex);
    const placeholder = isDisplay
      ? `<div data-math-display="${encoded}" class="math-display my-3 overflow-x-auto text-center"></div>`
      : `<span data-math="${encoded}" class="math-inline"></span>`;
    html = html.split(token).join(placeholder);
  }
  return html;
}

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

  // 1 — strip control chars that our placeholder tokens rely on, then mask
  //     code regions (spans + fenced/indented blocks) so math inside code
  //     stays literal.
  const cleaned = String(text)
    .replace(/\r\n?/g, '\n')
    .replace(/[\u0000-\u0008\u000b-\u001f]/g, '');
  const { text: codeMasked, restore } = maskCodeRegions(cleaned);

  // 2 — extract math BEFORE HTML-escaping so LaTeX syntax is preserved verbatim.
  const { text: mathStripped, stash: mathStash } = extractMath(codeMasked);

  const normalized = escapeHtml(mathStripped.replace(/\t/g, '    '));

  const lines = extractDefinitions(normalized.split('\n'), ctx);

  // 3 — link reference destinations end up inside href/title attributes; a
  //     raw math token there would corrupt the attribute, so blank them out.
  const attrSafe = (u: string) => u.replace(/\u0004M\d+\u0004/g, '');
  ctx.links.forEach((def) => {
    def.href = attrSafe(def.href);
    if (def.title) def.title = attrSafe(def.title);
  });

  let html = parseBlocks(lines, ctx) + renderFootnotes(ctx);

  // 4 — scrub any token that landed inside an attribute (href, title, id).
  html = html.replace(/((?:href|title|id|src|alt)="[^"]*)\u0004M\d+\u0004([^"]*")/g, '$1$2');

  // 5 — re-inject math placeholders into the final HTML.
  html = reinjectMath(html, mathStash);

  // 6 — restore masked code regions.
  return restore(html);
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
