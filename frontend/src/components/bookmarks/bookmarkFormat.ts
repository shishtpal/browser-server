import type { BookmarkResponse } from '../../types';

export type BookmarkSearchColumn = 'title' | 'url' | 'description' | 'folder' | 'tags' | 'all';

export const SEARCH_COLUMNS: { value: BookmarkSearchColumn; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'title', label: 'Title' },
  { value: 'url', label: 'URL' },
  { value: 'description', label: 'Description' },
  { value: 'folder', label: 'Folder' },
  { value: 'tags', label: 'Tags' },
];

export const SEARCH_PLACEHOLDERS: Record<BookmarkSearchColumn, string> = {
  all: 'Search bookmarks...',
  title: 'Search by title...',
  url: 'Search by URL...',
  description: 'Search description...',
  folder: 'Search folder path...',
  tags: 'Search tags...',
};

/** Avatar fallback letter for a bookmark title. */
export function getInitial(value: string): string {
  return value.trim().charAt(0).toUpperCase() || 'B';
}

/** Host part of a URL; falls back to the raw string if unparseable. */
export function formatHost(url: string): string {
  try {
    return new URL(url).host;
  } catch {
    return url;
  }
}

/** "a, b , c" → ["a", "b", "c"] */
export function parseTags(tagsStr: string): string[] {
  return tagsStr
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean);
}

/** Does a bookmark match ALL search terms in the given column? (space-separated) */
export function matchesSearch(
  b: BookmarkResponse,
  col: BookmarkSearchColumn,
  term: string,
): boolean {
  if (col === 'title') return b.title.toLowerCase().includes(term);
  if (col === 'url') return b.url.toLowerCase().includes(term);
  if (col === 'description') return (b.description || '').toLowerCase().includes(term);
  if (col === 'folder') return (b.folder_path || '').toLowerCase().includes(term);
  if (col === 'tags') return b.tags.some((t) => t.toLowerCase().includes(term));
  return [b.title, b.url, b.description, b.folder_path, ...b.tags]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
    .includes(term);
}
