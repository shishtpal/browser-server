import type { AIMessage } from '@browser-server/shared-types';

/** Shared chat formatting/derivation helpers (single source of truth). */

/** formatBytes — file sizes like "128 B" / "12.4 KB" / "3.2 MB". */
export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

/** Relative time: "just now" / "5m ago" / "3h ago" / "2d ago" / date. */
export function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return 'just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(iso).toLocaleDateString();
}

/** Short absolute timestamp ("Mar 4, 12:05") for message cards. */
export function formatMessageTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

/** Filesystem-safe filename for the conversation export. */
/** Filesystem-safe filename for the conversation export. */
export function filenameSafe(value: string, fallback = 'ai-conversation'): string {
  const illegal = new RegExp('[<>:"/\\|?*\x00-\x1F]', 'g');
  return (
    value
      .replace(illegal, '-')
      .replace(/[. ]+$/g, '')
      .slice(0, 100) || fallback
  );
}

/**
 * Prompt-library payloads historically shipped the text under several shapes
 * (`content`, `Content`, `prompt.content`, `Prompt.content`); normalize to a
 * plain string so textareas never render "[object Object]".
 */
export function normalizePromptContent(value: unknown): string {
  if (typeof value === 'string') return value;
  if (value == null) return '';
  const v = value as Record<string, any>;
  const candidate = v.content ?? v.Content ?? v.Prompt?.content ?? v.prompt?.content;
  if (typeof candidate === 'string') return candidate;
  if (candidate == null) return '';
  try {
    return JSON.stringify(candidate);
  } catch {
    return String(candidate);
  }
}

/** Truncate long message content for preview cards. */
export function truncateContent(content: string, maxLength = 500): string {
  if (content.length <= maxLength) return content;
  return content.slice(0, maxLength) + '…';
}

/** Role badge classes used by the memory explorer cards. */
export function roleBadgeClass(role: string): string {
  switch (role) {
    case 'user':
      return 'bg-slate-900 text-white dark:bg-white dark:text-slate-900';
    case 'assistant':
      return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300';
    case 'tool':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300';
    case 'system':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300';
    default:
      return 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300';
  }
}

/** Card border/background classes based on message state. */
export function messageBorderClass(msg: AIMessage, deleteTargetId: string | null): string {
  if (deleteTargetId === msg.id)
    return 'border-red-300 bg-red-50/80 dark:border-red-800/60 dark:bg-red-950/20';
  if (msg.status === 'error')
    return 'border-red-200 bg-red-50/50 dark:border-red-900/30 dark:bg-red-950/10';
  if (msg.status === 'superseded')
    return 'border-slate-200 bg-slate-50/50 opacity-50 dark:border-white/5 dark:bg-slate-900/30';
  return 'border-slate-200 bg-white dark:border-white/10 dark:bg-slate-900';
}
