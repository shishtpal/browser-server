import { adminHeaders } from '../auth';
import { API_BASE } from './client';

export type ConfigClass = 'leaf' | 'core';
export type ReloadSemantics = 'hot_reload' | 'restart_required';

export interface AdminStatus {
  managed: boolean;
  admin_configured: boolean;
  uptime_seconds: number;
}

export interface AdminConfigFile {
  name: string;
  class: ConfigClass;
  reload: ReloadSemantics;
  exists: boolean;
  size: number;
  modified_at?: string;
}

export interface AdminConfigContent {
  name: string;
  class: ConfigClass;
  reload: ReloadSemantics;
  content: string;
}

export interface AdminConfigMutation {
  saved?: boolean;
  reloaded?: boolean;
  reload: 'hot_reloaded' | 'restart_required' | 'failed';
  warning?: string;
  restart_required?: boolean;
  error?: string;
}

export class AdminAPIError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = 'AdminAPIError';
  }
}

function errorDetails(status: number, body: unknown): AdminAPIError {
  let code = `http_${status}`;
  let message = `Admin request failed with status ${status}`;
  if (body && typeof body === 'object' && 'error' in body) {
    const error = (body as { error: unknown }).error;
    if (typeof error === 'string') {
      code = error;
      const topLevelMessage = (body as { message?: unknown }).message;
      message =
        typeof topLevelMessage === 'string'
          ? topLevelMessage
          : error === 'admin_disabled'
            ? 'The admin API is not configured.'
            : error;
    } else if (error && typeof error === 'object') {
      const nested = error as { code?: unknown; message?: unknown };
      if (typeof nested.code === 'string') code = nested.code;
      if (typeof nested.message === 'string') message = nested.message;
    }
  }
  return new AdminAPIError(status, code, message);
}

async function fetchJSON<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      ...adminHeaders(),
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...init.headers,
    },
  });
  const text = await response.text();
  let body: unknown = undefined;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = { error: text };
    }
  }
  if (!response.ok) throw errorDetails(response.status, body);
  return body as T;
}

export function getAdminStatus(): Promise<AdminStatus> {
  return fetchJSON('/api/admin/status');
}

export function listAdminConfigFiles(): Promise<AdminConfigFile[]> {
  return fetchJSON('/api/admin/config/files');
}

export function getAdminConfigFile(name: string): Promise<AdminConfigContent> {
  return fetchJSON(`/api/admin/config/files/${encodeURIComponent(name)}`);
}

export function putAdminConfigFile(name: string, content: string): Promise<AdminConfigMutation> {
  return fetchJSON(`/api/admin/config/files/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify({ content }),
  });
}

export function reloadAdminConfigFile(name: string): Promise<AdminConfigMutation> {
  return fetchJSON(`/api/admin/config/reload/${encodeURIComponent(name)}`, { method: 'POST' });
}

export function restartServer(): Promise<{ restarting: boolean }> {
  return fetchJSON('/api/admin/restart', { method: 'POST' });
}
