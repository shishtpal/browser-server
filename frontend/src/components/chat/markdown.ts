/**
 * Lightweight Markdown to HTML renderer.
 * Handles code blocks, inline code, bold, italic, headers, and lists.
 * Input is escaped first to prevent XSS.
 */

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

export function renderMarkdown(text: string): string {
  if (!text) return ''
  let html = escapeHtml(text)

  // Fenced code blocks
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, (_m, lang, code) =>
    `<pre class="rounded-lg bg-slate-100 p-3 text-xs overflow-x-auto dark:bg-slate-800"><code class="language-${lang}">${code.trim()}</code></pre>`
  )

  // Inline code
  html = html.replace(/`([^`]+)`/g, '<code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-mono dark:bg-slate-800">$1</code>')

  // Bold (before italic)
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')

  // Italic
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')

  // Headers
  html = html.replace(/^### (.+)$/gm, '<h4 class="font-bold mt-3 mb-1">$1</h4>')
  html = html.replace(/^## (.+)$/gm, '<h3 class="font-bold text-base mt-4 mb-1">$1</h3>')
  html = html.replace(/^# (.+)$/gm, '<h2 class="font-black text-lg mt-4 mb-2">$1</h2>')

  // Unordered lists
  html = html.replace(/^- (.+)$/gm, '<li class="ml-4 list-disc">$1</li>')

  // Ordered lists
  html = html.replace(/^\d+\. (.+)$/gm, '<li class="ml-4 list-decimal">$1</li>')

  // Paragraphs (double newline)
  html = html.replace(/\n\n/g, '</p><p class="mt-2">')

  // Single newlines
  html = html.replace(/\n/g, '<br/>')

  return `<p>${html}</p>`
}
