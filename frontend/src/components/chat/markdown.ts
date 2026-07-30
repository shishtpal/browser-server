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

  // Protect fenced code blocks from the remaining Markdown replacements.
  const codeBlocks: string[] = []
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, (_m, lang, code) => {
    const index = codeBlocks.push(
      `<div class="not-prose group/code relative my-3"><button type="button" data-copy-code class="absolute right-2 top-2 rounded-md bg-slate-200/90 px-2 py-1 text-[0.7em] font-semibold text-slate-600 transition hover:bg-slate-300 sm:opacity-0 sm:group-hover/code:opacity-100 sm:focus:opacity-100 dark:bg-slate-700/90 dark:text-slate-200 dark:hover:bg-slate-600" title="Copy code" aria-label="Copy code">Copy</button><pre class="overflow-x-auto rounded-lg bg-slate-100 p-3 pr-14 text-[0.85em] dark:bg-slate-800"><code class="language-${lang}">${code.trim()}</code></pre></div>`
    ) - 1
    return `\u0000CODE_BLOCK_${index}\u0000`
  })

  // Inline code
  html = html.replace(/`([^`]+)`/g, '<code class="rounded bg-slate-100 px-1.5 py-0.5 text-[0.85em] font-mono dark:bg-slate-800">$1</code>')

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

  html = html.replace(/\u0000CODE_BLOCK_(\d+)\u0000/g, (_match, index) => codeBlocks[Number(index)] ?? '')

  return `<p>${html}</p>`
}
