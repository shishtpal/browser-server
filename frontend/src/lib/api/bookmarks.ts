import type { Bookmark, BookmarkResponse, ImportResult } from '@browser-server/shared-types';
import { API_BASE, authHeaders } from './client';

export function getBookmarks(userId?: number, tags?: string): Promise<BookmarkResponse[]> {
  const params = new URLSearchParams();
  if (userId) params.set('user_id', String(userId));
  if (tags) params.set('tags', tags);
  const qs = params.toString();
  return fetch(`${API_BASE}/api/bookmarks${qs ? `?${qs}` : ''}`, { headers: authHeaders() }).then(
    (res) => {
      if (!res.ok) throw new Error(`Request failed: ${res.status}`);
      return res.json() as Promise<BookmarkResponse[]>;
    },
  );
}

export function getBookmark(id: number): Promise<BookmarkResponse> {
  return fetch(`${API_BASE}/api/bookmarks/${id}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`);
    return res.json() as Promise<BookmarkResponse>;
  });
}

export function createBookmark(data: {
  user_id: number;
  title: string;
  url: string;
  description?: string;
  tags?: string[];
}): Promise<BookmarkResponse> {
  return fetch(`${API_BASE}/api/bookmarks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) {
      return res.text().then((text) => {
        throw new Error(text || `Request failed: ${res.status}`);
      });
    }
    return res.json() as Promise<BookmarkResponse>;
  });
}

export function updateBookmark(id: number, data: Partial<Bookmark>): Promise<BookmarkResponse> {
  return fetch(`${API_BASE}/api/bookmarks/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`);
    return res.json() as Promise<BookmarkResponse>;
  });
}

export function deleteBookmark(id: number): Promise<void> {
  return fetch(`${API_BASE}/api/bookmarks/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  }).then((res) => {
    if (res.status === 204) return;
    if (!res.ok) throw new Error(`Request failed: ${res.status}`);
  });
}

export function importBookmarks(userId: number, file: File): Promise<ImportResult> {
  const formData = new FormData();
  formData.append('file', file);
  return fetch(`${API_BASE}/api/bookmarks/import?user_id=${userId}`, {
    method: 'POST',
    headers: authHeaders(),
    body: formData,
  }).then(async (res) => {
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || `Import failed: ${res.status}`);
    }
    return res.json();
  });
}
