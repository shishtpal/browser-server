/** Wallet search columns + display helpers (single source of truth). */

export type WalletSearchColumn = 'website' | 'login_provider' | 'username' | 'description' | 'all';

export const WALLET_SEARCH_COLUMNS: { value: WalletSearchColumn; label: string }[] = [
  { value: 'website', label: 'Website' },
  { value: 'login_provider', label: 'Login provider' },
  { value: 'username', label: 'Username' },
  { value: 'description', label: 'Description' },
  { value: 'all', label: 'All columns' },
];

export const WALLET_SEARCH_PLACEHOLDERS: Record<WalletSearchColumn, string> = {
  website: 'Search by website URL...',
  login_provider: 'Search by login provider...',
  username: 'Search by username...',
  description: 'Search description...',
  all: 'Search all columns...',
};

/** Avatar fallback letter for a website name. */
export function walletInitial(value: string): string {
  return value.trim().charAt(0).toUpperCase() || 'W';
}

/** Provider-based logins (Google/GitHub SSO…) don't store a password. */
export function isPasswordless(provider: string): boolean {
  return provider.trim().toLowerCase() !== 'password';
}
