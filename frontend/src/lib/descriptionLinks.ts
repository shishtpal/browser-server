const URL_RE = /(https?:\/\/[^\s<>"]+)/g
const SAFE_SCHEMES = /^(https?):\/\//i

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

export function linkifyDescription(text: string): string {
  if (!text) return ''
  return escapeHtml(text).replace(URL_RE, (match) => {
    if (!SAFE_SCHEMES.test(match)) return match
    return `<a href="${match}" target="_blank" rel="noopener noreferrer" class="underline text-blue-600 dark:text-blue-400 hover:opacity-80">${match}</a>`
  })
}

export function hasLink(text: string | null | undefined): boolean {
  if (!text) return false
  return /https?:\/\//.test(text)
}