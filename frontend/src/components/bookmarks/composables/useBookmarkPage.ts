import type { BookmarkResponse } from '../../../types';
import { ref, type Ref } from 'vue';
import { useModal } from '@browser-server/shared-modal';
import { useBookmarks } from './useBookmarks';
import { useBookmarkTree } from './useBookmarkTree';

/**
 * Orchestrates the Bookmarks page: tag filtering, flat/tree view switching,
 * the edit modal and delete confirmation — so BookmarkPage.vue stays pure wiring.
 * Domain state lives in useBookmarks / useBookmarkTree.
 */
export function useBookmarkPage(userId: Ref<number | null>) {
  const bookmarksApi = useBookmarks(userId);

  const tree = useBookmarkTree(bookmarksApi.filteredBookmarks, bookmarksApi.searchQuery);

  /* ------------------------------- edit modal ------------------------------ */

  const editing = ref<BookmarkResponse | null>(null);

  const openEdit = (bookmark: BookmarkResponse) => {
    editing.value = bookmark;
  };

  const closeEdit = () => {
    editing.value = null;
  };

  const handleSaveEdit = async (data: {
    title: string;
    url: string;
    description: string;
    tagsStr: string;
  }) => {
    if (!editing.value) return;
    const saved = await bookmarksApi.editBookmark(editing.value.id, data);
    if (saved) closeEdit();
  };

  /* --------------------------- delete confirmation -------------------------- */

  const { confirmDelete: confirmDeleteModal } = useModal();

  async function confirmDelete(id: number) {
    const bookmark = bookmarksApi.bookmarks.value.find((b) => b.id === id);
    const confirmed = await confirmDeleteModal(
      `Delete "${bookmark?.title || 'this bookmark'}"?`,
      'This action cannot be undone.',
    );
    if (confirmed) await bookmarksApi.removeBookmark(id);
  }

  return {
    bookmarksApi,
    ...tree,
    editing,
    openEdit,
    closeEdit,
    handleSaveEdit,
    confirmDelete,
  };
}
