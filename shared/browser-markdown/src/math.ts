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

export interface MathStash {
  token: string;
  isDisplay: boolean;
  latex: string;
}

/** Placeholder tokens use \u0004 … \u0004 (EOT), which never appears in real
 *  input (we strip it up front along with \u0000), so user text can never
 *  collide with a generated token. */
export const MATH_TOKEN_PREFIX = '\u0004M';
export const MATH_TOKEN_SUFFIX = '\u0004';
export const CODE_TOKEN_PREFIX = '\u0004C';

/**
 * Mask regions where math must NOT be extracted (code spans, fenced code
 * blocks, indented code) so $…$ inside code stays literal, and decode them
 * back afterwards. Tokens survive HTML-escaping untouched.
 */
export function maskCodeRegions(text: string): { text: string; restore: (s: string) => string } {
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
export function extractMath(text: string): { text: string; stash: MathStash[] } {
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
export function reinjectMath(html: string, stash: MathStash[]): string {
  for (const { token, isDisplay, latex } of stash) {
    const encoded = encodeURIComponent(latex);
    const placeholder = isDisplay
      ? `<div data-math-display="${encoded}" class="math-display my-3 overflow-x-auto text-center"></div>`
      : `<span data-math="${encoded}" class="math-inline"></span>`;
    html = html.split(token).join(placeholder);
  }
  return html;
}
