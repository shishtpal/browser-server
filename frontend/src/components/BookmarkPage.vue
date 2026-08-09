<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Saved links" title="Bookmarks" color="cyan">
      <template #stats>
        <StatCard :value="bookmarks.length" label="Saved" variant="dark" color="cyan" />
        <StatCard :value="allTags.length" label="Tags" variant="primary" color="cyan" />
      </template>
      <template #controls>
        <UserSelector id="bookmark-user" v-model="selectedUserId" :users="users" color="cyan" />
      </template>
      <template v-if="selectedUserId" #actions>
        <BookmarkTagFilter
          :all-tags="allTags"
          :active-tag="activeTagFilter"
          @select="filterByTag"
          @clear="clearTagFilter"
        />
      </template>
    </PageHeader>

    <SelectUserPrompt
      title="Select a user to manage their bookmarks"
      :users-count="users.length"
      :selected-user-id="selectedUserId"
    />

    <LoadingSpinner v-if="isLoading" message="Loading bookmarks..." color="cyan" />
    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadBookmarks" />

    <div v-else-if="selectedUserId">
      <BookmarkForm @submit="addBookmark" />

      <BookmarkImport :selected-user-id="selectedUserId" class="mt-3" @imported="loadBookmarks" />

      <BookmarkSearchBar
        v-model:search-query="searchQuery"
        v-model:search-column="searchColumn"
        v-model:view-mode="viewMode"
        :filtered-count="filteredBookmarks.length"
        :tree-count="treeCount"
        :total-count="bookmarks.length"
      />

      <EmptyState
        v-if="bookmarks.length === 0"
        title="No bookmarks yet"
        description="Save your first link above."
        icon="bookmark"
        color="cyan"
      />

      <EmptyState
        v-else-if="filteredBookmarks.length === 0"
        title="No results match your search"
        description="Try a different search term or clear the filter."
        icon="search"
        color="cyan"
      />

      <BookmarkFlatView
        v-else-if="viewMode === 'flat'"
        :bookmarks="filteredBookmarks"
        @edit="openEdit"
        @delete="confirmDelete"
        @filter-tag="filterByTag"
      />

      <BookmarkTreeView
        v-else
        :nodes="treeNodes"
        @toggle-folder="toggleTreeFolder"
        @edit="openEdit"
        @delete="confirmDelete"
        @filter-tag="filterByTag"
      />
    </div>

    <BookmarkEditModal :bookmark="editing" @close="closeEdit" @save="handleSaveEdit" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useUser } from '../composables/useUser';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import UserSelector from './ui/UserSelector.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import EmptyState from './ui/EmptyState.vue';
import SelectUserPrompt from './ui/SelectUserPrompt.vue';
import { useBookmarkPage } from './bookmarks/composables/useBookmarkPage';
import BookmarkForm from './bookmarks/BookmarkForm.vue';
import BookmarkImport from './bookmarks/BookmarkImport.vue';
import BookmarkSearchBar from './bookmarks/BookmarkSearchBar.vue';
import BookmarkTagFilter from './bookmarks/BookmarkTagFilter.vue';
import BookmarkFlatView from './bookmarks/views/BookmarkFlatView.vue';
import BookmarkTreeView from './bookmarks/BookmarkTreeView.vue';
import BookmarkEditModal from './bookmarks/BookmarkEditModal.vue';

const { users, currentUserId, setUser, clearUser } = useUser();
const selectedUserId = ref<number | null>(currentUserId.value);

const {
  bookmarksApi: {
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
  },
  viewMode,
  treeNodes,
  treeCount,
  toggleTreeFolder,
  editing,
  openEdit,
  closeEdit,
  handleSaveEdit,
  confirmDelete,
} = useBookmarkPage(selectedUserId);

watch(selectedUserId, (id) => {
  clearTagFilter();
  if (id) setUser(id);
  else clearUser();
});

// Data loads automatically via the immediate watcher inside useBookmarks.
</script>
