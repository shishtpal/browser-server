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
  /**
   * Extract LaTeX math ($…$, $$…$$, \(…\), \[…\]) into placeholder elements
   * for later `typesetMath()`. When `false`, math delimiters render as
   * literal text. Default: `true`.
   */
  math?: boolean;
  /** Override any of the default Tailwind class strings. */
  classes?: Partial<Record<StyleKey, string>>;
}

/* -------------------------------------------------------------- structure */

export interface Ctx {
  opts: Required<Omit<MarkdownOptions, 'classes' | 'math'>>;
  cls: Record<StyleKey, string>;
  links: Map<string, { href: string; title?: string }>;
  footnoteDefs: Map<string, string[]>;
  footnoteOrder: string[];
  slugs: Map<string, number>;
}
