import type { WalletEntry, WalletImportResult } from '@browser-server/shared-types';
import { API_BASE, authHeaders } from './client';

export function getWallet(userId?: number, website?: string): Promise<WalletEntry[]> {
  const params = new URLSearchParams();
  if (userId) params.set('user_id', String(userId));
  if (website) params.set('website', website);
  const qs = params.toString();
  return fetch(`${API_BASE}/api/wallet${qs ? `?${qs}` : ''}`, { headers: authHeaders() }).then(
    (res) => {
      if (!res.ok) throw new Error(`Request failed: ${res.status}`);
      return res.json() as Promise<WalletEntry[]>;
    },
  );
}

export function revealWalletPassword(userId: number, id: number): Promise<string> {
  const params = new URLSearchParams({ user_id: String(userId), id: String(id) });
  return fetch(`${API_BASE}/api/wallet/reveal?${params.toString()}`, {
    headers: authHeaders(),
  }).then(async (res) => {
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || `Request failed: ${res.status}`);
    }
    return ((await res.json()) as { password: string }).password;
  });
}

export function importWallet(userId: number, file: File): Promise<WalletImportResult> {
  const formData = new FormData();
  formData.append('file', file);
  return fetch(`${API_BASE}/api/wallet/import?user_id=${userId}`, {
    method: 'POST',
    headers: authHeaders(),
    body: formData,
  }).then(async (res) => {
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || `Import failed: ${res.status}`);
    }
    return res.json() as Promise<WalletImportResult>;
  });
}

export function getWalletEntry(id: number): Promise<WalletEntry> {
  return fetch(`${API_BASE}/api/wallet/${id}`, { headers: authHeaders() }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`);
    return res.json() as Promise<WalletEntry>;
  });
}

export function createWalletEntry(data: {
  user_id: number;
  website: string;
  username: string;
  password: string;
  login_provider?: string;
  description?: string;
}): Promise<WalletEntry> {
  return fetch(`${API_BASE}/api/wallet`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`);
    return res.json() as Promise<WalletEntry>;
  });
}

export function updateWalletEntry(id: number, data: Partial<WalletEntry>): Promise<WalletEntry> {
  return fetch(`${API_BASE}/api/wallet/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...authHeaders() },
    body: JSON.stringify(data),
  }).then((res) => {
    if (!res.ok) throw new Error(`Request failed: ${res.status}`);
    return res.json() as Promise<WalletEntry>;
  });
}

export function deleteWalletEntry(id: number): Promise<void> {
  return fetch(`${API_BASE}/api/wallet/${id}`, { method: 'DELETE', headers: authHeaders() }).then(
    (res) => {
      if (res.status === 204) return;
      if (!res.ok) throw new Error(`Request failed: ${res.status}`);
    },
  );
}
