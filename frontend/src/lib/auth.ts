// Client-side storage for the disjoint operator and administrator API tokens.
// The operator token is attached to ordinary API requests; the stronger admin
// token is used only by the Project Settings API module.

const TOKEN_KEY = 'api_token';
export const ADMIN_TOKEN_KEY = 'api_token_admin';

export function getToken(): string | null {
  if (typeof localStorage === 'undefined') return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  if (typeof localStorage === 'undefined') return;
  const trimmed = token.trim();
  if (trimmed) {
    localStorage.setItem(TOKEN_KEY, trimmed);
  } else {
    localStorage.removeItem(TOKEN_KEY);
  }
  window.dispatchEvent(new CustomEvent('api-token-changed'));
}

export function clearToken(): void {
  if (typeof localStorage === 'undefined') return;
  localStorage.removeItem(TOKEN_KEY);
  window.dispatchEvent(new CustomEvent('api-token-changed'));
}

export function hasToken(): boolean {
  return Boolean(getToken());
}

/** Authorization header object for raw operator fetch calls. */
export function authHeaders(): Record<string, string> {
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

export function getAdminToken(): string | null {
  if (typeof localStorage === 'undefined') return null;
  return localStorage.getItem(ADMIN_TOKEN_KEY);
}

export function setAdminToken(token: string): void {
  if (typeof localStorage === 'undefined') return;
  const trimmed = token.trim();
  if (trimmed) {
    localStorage.setItem(ADMIN_TOKEN_KEY, trimmed);
  } else {
    localStorage.removeItem(ADMIN_TOKEN_KEY);
  }
  window.dispatchEvent(new CustomEvent('api-admin-token-changed'));
}

export function clearAdminToken(): void {
  if (typeof localStorage === 'undefined') return;
  localStorage.removeItem(ADMIN_TOKEN_KEY);
  window.dispatchEvent(new CustomEvent('api-admin-token-changed'));
}

export function hasAdminToken(): boolean {
  return Boolean(getAdminToken());
}

/** Authorization header object used exclusively by administrator endpoints. */
export function adminHeaders(): Record<string, string> {
  const token = getAdminToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
