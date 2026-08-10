/* ------------------------------------------------------------------ utils */

export const NUL = '\u0000';

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
export function escapeAttr(str: string): string {
  return str.replace(/[^\w.:-]/g, '-');
}

const SAFE_SCHEMES = /^(?:https?|mailto|tel|sms|ftp):/i;

/** Input is already HTML-escaped; we only need to police the scheme. */
export function sanitizeUrl(raw: string): string {
  const url = raw.trim().replace(/^&lt;/, '').replace(/&gt;$/, '');
  const probe = url.replace(/[\s\u0000-\u001f]/g, '').toLowerCase();
  if (/^data:image\/(?:png|jpe?g|gif|webp|avif|svg\+xml);base64,/.test(probe)) return url;
  if (/^[a-z][a-z0-9+.-]*:/.test(probe) && !SAFE_SCHEMES.test(probe)) return '#';
  return url;
}

export function slugify(text: string, taken: Map<string, number>): string {
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
