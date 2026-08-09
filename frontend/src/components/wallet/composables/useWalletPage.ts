import type { WalletEntry } from '../../../types';
import { computed, ref, type Ref } from 'vue';
import { formatDate } from '../../../lib/utils';
import { WALLET_SEARCH_PLACEHOLDERS } from '../walletFormat';
import { useWallet, type WalletEntryInput } from './useWallet';
import { useModal } from '@browser-server/shared-modal';

/** The payload shape the edit modal works with (form strings + id). */
export interface WalletEditDraft extends WalletEntryInput {
  id: number;
}

/**
 * Orchestrates the Wallet page: edit modal (with on-demand password
 * prefill for the entry being edited) and delete confirmation — so
 * WalletPage.vue stays pure wiring.
 */
export function useWalletPage(userId: Ref<number | null>) {
  const walletApi = useWallet(userId);

  /* ------------------------------- edit modal ------------------------------ */

  const editing = ref<WalletEntry | null>(null);
  /** The revealed password of the entry being edited ("" until fetched). */
  const editingPassword = ref('');
  const isRevealingPassword = ref(false);

  const openEdit = async (entry: WalletEntry) => {
    editing.value = entry;
    editingPassword.value = '';
    isRevealingPassword.value = true;
    try {
      editingPassword.value = await walletApi.revealPassword(entry.id);
    } catch {
      editingPassword.value = '';
    } finally {
      isRevealingPassword.value = false;
    }
  };

  const closeEdit = () => {
    editing.value = null;
    editingPassword.value = '';
  };

  const saveEdit = async (input: WalletEntryInput) => {
    if (!editing.value) return;
    const saved = await walletApi.saveEntry(editing.value.id, input);
    if (saved) closeEdit();
  };

  /* --------------------------- delete confirmation -------------------------- */

  const { confirmDelete: confirmDeleteModal } = useModal();

  async function confirmDelete(id: number) {
    const entry = walletApi.walletEntries.value.find((e) => e.id === id);
    const confirmed = await confirmDeleteModal(
      `Delete "${entry?.website || 'this entry'}" credentials?`,
      'This action cannot be undone. The saved login will be permanently removed.',
    );
    if (confirmed) await walletApi.removeEntry(id);
  }

  /* ------------------------------ display bits ----------------------------- */

  const entryTimestamp = (entry: WalletEntry) => formatDate(entry.updated_at || '');

  const searchPlaceholder = computed(
    () => WALLET_SEARCH_PLACEHOLDERS[walletApi.searchColumn.value] ?? 'Search...',
  );

  return {
    walletApi,
    editing,
    editingPassword,
    isRevealingPassword,
    openEdit,
    closeEdit,
    saveEdit,
    confirmDelete,
    entryTimestamp,
    searchPlaceholder,
  };
}
