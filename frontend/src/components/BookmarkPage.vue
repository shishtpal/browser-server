<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Saved links" title="Bookmarks" color="cyan">
      <template #stats>
        <StatCard :value="bookmarks.length" label="Saved" variant="dark" color="cyan" />
        <StatCard :value="allTags.length" label="Tags" variant="primary" color="cyan" />
      </template>
      <template #actions>
        <UserSelector id="bookmark-user" v-model="selectedUserId" :users="users" color="cyan" />
        <div v-if="allTags.length" class="flex flex-wrap gap-1">
          <FilterPill :active="activeTagFilter === null" @click="activeTagFilter = null">All</FilterPill>
          <FilterPill
            v-for="tag in allTags"
            :key="tag"
            :active="activeTagFilter === tag"
            variant="tag"
            @click="filterByTag(tag)"
          >
            #{{ tag }}
          </FilterPill>
        </div>
      </template>
    </PageHeader>

    <SelectUserPrompt title="Select a user to manage their bookmarks" :users-count="users.length" :selected-user-id="selectedUserId" />

    <LoadingSpinner v-if="isLoading" message="Loading bookmarks..." color="cyan" />

    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadBookmarks" />

    <div v-else-if="selectedUserId">
      <BookmarkForm
        v-model:new-title="newTitle"
        v-model:new-url="newUrl"
        v-model:new-description="newDescription"
        v-model:new-tags="newTags"
        @submit="addBookmark"
      />

      <BookmarkImport
        v-if="selectedUserId"
        :selected-user-id="selectedUserId"
        class="mt-3"
        @imported="loadBookmarks"
      />

      <BookmarkSearchBar
        v-model:search-query="searchQuery"
        v-model:search-column="searchColumn"
        v-model:view-mode="viewMode"
        :filtered-count="filteredBookmarks.length"
        :tree-count="treeCount"
        :total-count="bookmarks.length"
      />

      <div v-if="activeTagFilter" class="mb-4 flex items-center gap-2 rounded-xl border border-cyan-200 bg-cyan-50/80 p-2 text-xs text-cyan-800 shadow-sm transition-colors dark:border-cyan-900/30 dark:bg-cyan-900/20 dark:text-cyan-300">
        <span class="font-bold">Filtering by tag:</span>
        <span class="rounded-md bg-white px-2 py-0.5 font-black text-cyan-700 shadow-sm transition-colors dark:bg-slate-800 dark:text-cyan-400">{{ activeTagFilter }}</span>
        <button type="button" @click="activeTagFilter = null" class="rounded-md bg-cyan-200 px-2 py-0.5 font-black text-cyan-800 transition hover:bg-cyan-300 dark:bg-cyan-800 dark:text-cyan-200 dark:hover:bg-cyan-700">Clear</button>
      </div>

      <div>
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
          color="amber"
        />

        <BookmarkFlatView
          v-else-if="viewMode === 'flat'"
          :bookmarks="filteredBookmarks"
          @edit="openEdit"
          @delete="removeBookmark"
          @filter-tag="filterByTag"
        />

        <BookmarkTreeView
          v-else
          :nodes="treeNodes"
          @toggle-folder="toggleTreeFolder"
          @edit="openEdit"
          @delete="removeBookmark"
          @filter-tag="filterByTag"
        />
      </div>
    </div>

    <div v-else class="rounded-xl border border-dashed border-gray-300 bg-gray-50 p-6 text-center shadow-sm transition-colors dark:border-slate-600 dark:bg-slate-800/60">
      <div class="mx-auto grid h-10 w-10 place-items-center rounded-xl bg-gray-100 text-slate-400 transition-colors dark:bg-slate-700 dark:text-slate-500">
        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
      <h2 class="mt-3 text-base font-black text-slate-800 transition-colors dark:text-slate-200">Choose a workspace</h2>
      <p class="mt-1 text-xs text-slate-500 transition-colors dark:text-slate-400">Select a user to manage their bookmarks.</p>
    </div>

    <BookmarkEditModal
      :bookmark="editing"
      @close="editing = null"
      @save="handleSaveEdit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useUser } from '../composables/useUser'
import { useBookmarks } from '../composables/useBookmarks'
import { useBookmarkTree } from '../composables/useBookmarkTree'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import UserSelector from './ui/UserSelector.vue'
import FilterPill from './ui/FilterPill.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import EmptyState from './ui/EmptyState.vue'
import SelectUserPrompt from './ui/SelectUserPrompt.vue'
import BookmarkForm from './bookmarks/BookmarkForm.vue'
import BookmarkImport from './bookmarks/BookmarkImport.vue'
import BookmarkSearchBar from './bookmarks/BookmarkSearchBar.vue'
import BookmarkFlatView from './bookmarks/BookmarkFlatView.vue'
import BookmarkTreeView from './bookmarks/BookmarkTreeView.vue'
import BookmarkEditModal from './bookmarks/BookmarkEditModal.vue'

const { users, currentUserId, setUser, clearUser } = useUser()

const selectedUserId = ref<number | null>(currentUserId.value)

const {
  bookmarks,
  isLoading,
  error,
  activeTagFilter,
  searchQuery,
  searchColumn,
  newTitle,
  newUrl,
  newDescription,
  newTags,
  editing,
  editForm,
  allTags,
  filteredBookmarks,
  loadBookmarks,
  addBookmark,
  filterByTag,
  openEdit,
  saveEdit,
  removeBookmark,
} = useBookmarks(selectedUserId)

const {
  viewMode,
  treeNodes,
  treeCount,
  toggleTreeFolder,
} = useBookmarkTree(filteredBookmarks, searchQuery)

const handleSaveEdit = (data: { title: string; url: string; description: string; tagsStr: string }) => {
  editForm.value = data
  saveEdit()
}

watch(selectedUserId, (id) => {
  activeTagFilter.value = null
  if (id) {
    setUser(id)
    loadBookmarks()
  } else {
    clearUser()
    bookmarks.value = []
  }
})

if (selectedUserId.value) {
  setUser(selectedUserId.value)
  loadBookmarks()
}
</script>
