import type { AIMessage } from '@browser-server/shared-types'

/** Parse the JSON content envelope of a tool message. */
export function parseToolContent(msg: AIMessage): { tool?: string; args?: unknown; result?: unknown; decision?: string } {
  try {
    return JSON.parse(msg.content)
  } catch {
    return {}
  }
}

/** Display-friendly tool name from snake_case. */
export function getToolName(msg: AIMessage): string {
  const parsed = parseToolContent(msg)
  if (!parsed.tool) return ''
  return parsed.tool.split('_').filter(Boolean).map((w) => w[0].toUpperCase() + w.slice(1)).join(' ')
}

/** Pretty-printed tool arguments. */
export function getToolArgs(msg: AIMessage): string {
  const parsed = parseToolContent(msg)
  if (parsed.args === null || parsed.args === undefined) return ''
  if (typeof parsed.args === 'string') return parsed.args
  try { return JSON.stringify(parsed.args, null, 2) } catch { return String(parsed.args) }
}

/** Pretty-printed tool result. */
export function getToolResult(msg: AIMessage): string {
  const parsed = parseToolContent(msg)
  if (parsed.result === null || parsed.result === undefined) return ''
  if (typeof parsed.result === 'string') return parsed.result
  try { return JSON.stringify(parsed.result, null, 2) } catch { return String(parsed.result) }
}

/** Tool decision string (approved / rejected / commented). */
export function getToolDecision(msg: AIMessage): string {
  return parseToolContent(msg).decision || ''
}

/** Formatted short timestamp for message display. */
export function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  } catch { return iso }
}

/** Truncate long message content for preview. */
export function truncateContent(content: string, maxLength = 500): string {
  if (content.length <= maxLength) return content
  return content.slice(0, maxLength) + '…'
}

/** CSS classes for the role badge. */
export function roleBadgeClass(role: string): string {
  switch (role) {
    case 'user': return 'bg-slate-900 text-white dark:bg-white dark:text-slate-900'
    case 'assistant': return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
    case 'tool': return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
    case 'system': return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
    default: return 'bg-slate-100 text-slate-600'
  }
}

/** CSS classes for the message card border based on state. */
export function messageBorderClass(msg: AIMessage, deleteTargetId: string | null): string {
  if (deleteTargetId === msg.id) return 'border-red-300 bg-red-50/80 dark:border-red-800/60 dark:bg-red-950/20'
  if (msg.status === 'error') return 'border-red-200 bg-red-50/50 dark:border-red-900/30 dark:bg-red-950/10'
  if (msg.status === 'superseded') return 'border-slate-200 bg-slate-50/50 opacity-50 dark:border-white/5 dark:bg-slate-900/30'
  return 'border-slate-200 bg-white dark:border-white/10 dark:bg-slate-900'
}
