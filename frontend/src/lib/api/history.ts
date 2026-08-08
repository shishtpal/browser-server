import type {
  CreateHistoryInput,
  History,
  HistoryImportResult,
} from '@browser-server/shared-types';
import { API_BASE, authHeaders, client } from './client';

export function getHistory(
  userId?: number,
  url?: string,
  limit?: number,
  offset?: number,
): Promise<History[]> {
  return client.getHistory(userId, url, limit, offset);
}

export function getHistoryEntry(id: number): Promise<History> {
  return fetch(`${API_BASE}/api/history/${id}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`);
    return res.json() as Promise<History>;
  });
}

export function createHistory(data: CreateHistoryInput): Promise<History> {
  return client.createHistory(data);
}

export function deleteHistory(id: number): Promise<void> {
  return client.deleteHistory(id);
}

export function importHistory(userId: number, file: File): Promise<HistoryImportResult> {
  const formData = new FormData();
  formData.append('file', file);
  return fetch(`${API_BASE}/api/history/import?user_id=${userId}`, {
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
