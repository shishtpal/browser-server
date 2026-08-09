import type { WalletEntry } from '../../../types';
import { computed, ref, watch, type Ref } from 'vue';
import {
  createWalletEntry,
  deleteWalletEntry,
  getWallet,
  revealWalletPassword,
  updateWalletEntry,
} from '../../../lib/api';
import type { WalletSearchColumn } from '../walletFormat';

export interface WalletEntryInput {
  website: string;
  login_provider: string;
  username: string;
  password: string;
  description?: string;
}

/**
 * Wallet (password vault) state for the selected user: list, search filter.
 * CRUD + on-demand password reveal (passwords are never part of list payloads).
 *
 * Loading starts automatically (immediate watcher) whenever the user changes.
 * Delete confirmation is the page's job (shared modal); this layer just does
 * the API call.
 */
export function useWallet(selectedUserId: Ref<number | null>) {
  const walletEntries = ref<WalletEntry[]>([]);
  const isLoading = ref(false);
  const error = ref<string | null>(null);
  const searchQuery = ref('');
  const searchColumn = ref<WalletSearchColumn>('website');

  const filteredEntries = computed(() => {
    const q = searchQuery.value.trim().toLowerCase();
    if (!q) return walletEntries.value;
    const col = searchColumn.value;
    return walletEntries.value.filter((e) => {
      if (col === 'all') {
        return (
          e.website.toLowerCase().includes(q) ||
          e.login_provider.toLowerCase().includes(q) ||
          e.username.toLowerCase().includes(q) ||
          e.description.toLowerCase().includes(q)
        );
      }
      if (col === 'website') return e.website.toLowerCase().includes(q);
      if (col === 'login_provider') return e.login_provider.toLowerCase().includes(q);
      if (col === 'username') return e.username.toLowerCase().includes(q);
      if (col === 'description') return e.description.toLowerCase().includes(q);
      return false;
    });
  });

  const loadWallet = async () => {
    if (!selectedUserId.value) return;
    isLoading.value = true;
    error.value = null;
    try {
      walletEntries.value = await getWallet(selectedUserId.value);
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load wallet';
    } finally {
      isLoading.value = false;
    }
  };

  const addEntry = async (input: WalletEntryInput) => {
    const provider = input.login_provider.trim() || 'Password';
    if (!selectedUserId.value || !input.website.trim() || !input.username.trim()) {
      return undefined;
    }
    if (provider.toLowerCase() === 'password' && !input.password) return undefined;
    try {
      const created = await createWalletEntry({
        user_id: selectedUserId.value,
        website: input.website.trim(),
        login_provider: provider,
        username: input.username.trim(),
        password: input.password,
        description: input.description?.trim() || undefined,
      });
      await loadWallet();
      return created;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add entry';
      return undefined;
    }
  };

  const saveEntry = async (id: number, input: WalletEntryInput) => {
    const current = walletEntries.value.find((e) => e.id === id);
    if (!current) return undefined;
    try {
      const updated = await updateWalletEntry(id, {
        ...current,
        website: input.website.trim(),
        login_provider: input.login_provider.trim() || 'Password',
        username: input.username.trim(),
        // Empty password keeps the stored one (modal help text promises this).
        ...(input.password ? { password: input.password } : {}),
        description: input.description?.trim() ?? '',
      });
      await loadWallet();
      return updated;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update entry';
      return undefined;
    }
  };

  const removeEntry = async (id: number) => {
    try {
      await deleteWalletEntry(id);
      walletEntries.value = walletEntries.value.filter((e) => e.id !== id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete entry';
    }
  };

  /** Passwords are only ever fetched on demand, one entry at a time. */
  const revealPassword = (id: number): Promise<string> => {
    if (!selectedUserId.value) return Promise.reject(new Error('No user selected'));
    return revealWalletPassword(selectedUserId.value, id);
  };

  watch(
    selectedUserId,
    (id) => {
      if (id && id > 0) {
        loadWallet();
      } else {
        walletEntries.value = [];
        searchQuery.value = '';
      }
    },
    { immediate: true },
  );

  return {
    walletEntries,
    isLoading,
    error,
    searchQuery,
    searchColumn,
    filteredEntries,
    loadWallet,
    addEntry,
    saveEntry,
    removeEntry,
    revealPassword,
  };
}
