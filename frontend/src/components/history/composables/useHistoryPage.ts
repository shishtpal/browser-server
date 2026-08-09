import type { History } from '@browser-server/shared-types';
import { ref, type Ref } from 'vue';
import { useIntersectionObserver } from '@vueuse/core';
import { useModal } from '@browser-server/shared-modal';
import { useHistory } from './useHistory';

/**
 * Orchestrates the History page: infinite scroll, add form, delete
 * confirmation — so HistoryPage.vue stays pure wiring.
 * Domain state lives in useHistory.
 */
export function useHistoryPage(userId: Ref<number | null>) {
  const historyApi = useHistory(userId);

  /* ---------------------------- infinite scroll ---------------------------- */

  const scrollSentinel = ref<HTMLElement | null>(null);

  const { stop: stopObserving } = useIntersectionObserver(
    scrollSentinel,
    (entries) => {
      if (entries[0]?.isIntersecting) {
        historyApi.loadMore();
      }
    },
    { rootMargin: '200px' },
  );

  /* --------------------------- delete confirmation -------------------------- */

  const { confirmDelete: confirmDeleteModal } = useModal();

  async function confirmDelete(entry: History) {
    const confirmed = await confirmDeleteModal(
      `Delete "${entry.title || 'this entry'}"?`,
      'This action cannot be undone.',
    );
    if (confirmed) await historyApi.removeEntry(entry.id);
  }

  return {
    historyApi,
    scrollSentinel,
    stopObserving,
    confirmDelete,
  };
}
