import type { History } from '@browser-server/shared-types';
import { computed, ref, watch, type Ref } from 'vue';
import { formatDuration } from '../../../lib/utils';
import { createHistory, deleteHistory, getHistory } from '../../../lib/api';

const PAGE_SIZE = 100;

export interface HistoryCreateInput {
  url: string;
  title: string;
  duration?: number;
}

/**
 * Browsing history for the selected user: paged list (auto "load more"),
 * client-side URL/title filter, add + delete actions.
 *
 * Loading starts automatically (immediate watcher) whenever the user changes.
 */
export function useHistory(selectedUserId: Ref<number | null>) {
  const historyEntries = ref<History[]>([]);
  const isLoading = ref(false);
  const isLoadingMore = ref(false);
  const error = ref<string | null>(null);
  const urlFilter = ref('');
  const hasMore = ref(false);

  const totalDuration = computed(() =>
    formatDuration(historyEntries.value.reduce((sum, h) => sum + h.duration, 0)),
  );

  const filteredHistory = computed(() => {
    const q = urlFilter.value.trim().toLowerCase();
    if (!q) return historyEntries.value;
    return historyEntries.value.filter(
      (h) => h.url.toLowerCase().includes(q) || h.title.toLowerCase().includes(q),
    );
  });

  const loadHistory = async () => {
    if (!selectedUserId.value) return;
    isLoading.value = true;
    error.value = null;
    try {
      const batch = await getHistory(selectedUserId.value, undefined, PAGE_SIZE, 0);
      historyEntries.value = batch;
      hasMore.value = batch.length >= PAGE_SIZE;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load history';
    } finally {
      isLoading.value = false;
    }
  };

  const loadMore = async () => {
    if (!selectedUserId.value || isLoadingMore.value || !hasMore.value) return;
    isLoadingMore.value = true;
    try {
      const offset = historyEntries.value.length;
      const batch = await getHistory(selectedUserId.value, undefined, PAGE_SIZE, offset);
      historyEntries.value = [...historyEntries.value, ...batch];
      hasMore.value = batch.length >= PAGE_SIZE;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load more history';
    } finally {
      isLoadingMore.value = false;
    }
  };

  const addEntry = async (input: HistoryCreateInput) => {
    if (!selectedUserId.value || !input.url.trim() || !input.title.trim()) return undefined;
    try {
      const created = await createHistory({
        user_id: selectedUserId.value,
        url: input.url.trim(),
        title: input.title.trim(),
        duration: input.duration ?? 0,
      });
      await loadHistory();
      return created;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add history entry';
      return undefined;
    }
  };

  /** Delete without confirming — the page confirms via the shared modal first. */
  const removeEntry = async (id: number) => {
    try {
      await deleteHistory(id);
      historyEntries.value = historyEntries.value.filter((h) => h.id !== id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete entry';
    }
  };

  watch(
    selectedUserId,
    (id) => {
      if (id && id > 0) {
        loadHistory();
      } else {
        historyEntries.value = [];
        urlFilter.value = '';
        hasMore.value = false;
      }
    },
    { immediate: true },
  );

  return {
    historyEntries,
    isLoading,
    isLoadingMore,
    error,
    urlFilter,
    hasMore,
    totalDuration,
    filteredHistory,
    loadHistory,
    loadMore,
    addEntry,
    removeEntry,
  };
}
