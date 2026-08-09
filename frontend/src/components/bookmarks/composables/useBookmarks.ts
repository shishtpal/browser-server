import type { Bookmark, BookmarkResponse } from '../../../types';
import { computed, ref, watch, type Ref } from 'vue';
import { createBookmark, deleteBookmark, getBookmarks, updateBookmark } from '../../../lib/api';
import { matchesSearch, parseTags, type BookmarkSearchColumn } from '../bookmarkFormat';

export interface BookmarkCreateInput {
  title: string;
  url: string;
  description?: string;
  tags?: string[];
}

/**
 * Bookmark list state for the selected user: tag filter, search, CRUD.
 *
 * Loading starts automatically (immediate watcher) whenever the user or the
 * active tag filter changes.
 */
export function useBookmarks(selectedUserId: Ref<number | null>) {
  const bookmarks = ref<BookmarkResponse[]>([]);
  const isLoading = ref(false);
  const error = ref<string | null>(null);
  const activeTagFilter = ref<string | null>(null);
  const searchQuery = ref('');
  const searchColumn = ref<BookmarkSearchColumn>('all');

  const allTags = computed(() =>
    Array.from(new Set(bookmarks.value.flatMap((b) => b.tags))).sort(),
  );

  const filteredBookmarks = computed(() => {
    const q = searchQuery.value.toLowerCase().trim();
    if (!q) return bookmarks.value;
    const col = searchColumn.value;
    const terms = q.split(/\s+/).filter(Boolean);
    return bookmarks.value.filter((b) => terms.every((t) => matchesSearch(b, col, t)));
  });

  const loadBookmarks = async () => {
    if (!selectedUserId.value) return;
    isLoading.value = true;
    error.value = null;
    try {
      bookmarks.value = await getBookmarks(
        selectedUserId.value,
        activeTagFilter.value || undefined,
      );
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load bookmarks';
    } finally {
      isLoading.value = false;
    }
  };

  const filterByTag = (tag: string) => {
    activeTagFilter.value = tag;
  };

  const clearTagFilter = () => {
    activeTagFilter.value = null;
  };

  const addBookmark = async (input: BookmarkCreateInput) => {
    if (!selectedUserId.value || !input.title.trim() || !input.url.trim()) return undefined;
    try {
      const created = await createBookmark({
        user_id: selectedUserId.value,
        title: input.title.trim(),
        url: input.url.trim(),
        description: input.description?.trim() || undefined,
        tags: input.tags?.length ? input.tags : undefined,
      });
      await loadBookmarks();
      return created;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add bookmark';
      return undefined;
    }
  };

  const editBookmark = async (
    id: number,
    input: { title: string; url: string; description: string; tagsStr: string },
  ) => {
    try {
      const current = bookmarks.value.find((b) => b.id === id);
      const updated = await updateBookmark(id, {
        user_id: current?.user_id ?? selectedUserId.value ?? 0,
        title: input.title.trim(),
        url: input.url.trim(),
        description: input.description.trim(),
        tags: parseTags(input.tagsStr),
      } as Partial<Bookmark>);
      await loadBookmarks();
      return updated;
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update bookmark';
      return undefined;
    }
  };

  /** Delete without confirming — the page confirms via the shared modal first. */
  const removeBookmark = async (id: number) => {
    try {
      await deleteBookmark(id);
      bookmarks.value = bookmarks.value.filter((b) => b.id !== id);
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete bookmark';
    }
  };

  watch(
    selectedUserId,
    (id) => {
      if (id && id > 0) {
        loadBookmarks();
      } else {
        bookmarks.value = [];
        activeTagFilter.value = null;
        searchQuery.value = '';
      }
    },
    { immediate: true },
  );

  watch(activeTagFilter, () => {
    if (selectedUserId.value) loadBookmarks();
  });

  return {
    bookmarks,
    isLoading,
    error,
    activeTagFilter,
    searchQuery,
    searchColumn,
    allTags,
    filteredBookmarks,
    loadBookmarks,
    filterByTag,
    clearTagFilter,
    addBookmark,
    editBookmark,
    removeBookmark,
  };
}
