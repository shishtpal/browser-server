<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Task manager" title="Todos" color="indigo">
      <template #stats>
        <StatCard :value="totalCount" label="Total" variant="dark" color="indigo" />
        <StatCard :value="activeCount" label="Active" variant="primary" color="indigo" />
        <StatCard
          v-if="inProgressCount > 0"
          :value="inProgressCount"
          label="In Progress"
          variant="primary"
          color="cyan"
        />
        <StatCard :value="completedCount" label="Done" variant="secondary" color="indigo" />
        <StatCard
          v-if="archivedCount > 0"
          :value="archivedCount"
          label="Archived"
          variant="secondary"
          color="indigo"
        />
        <StatCard
          v-if="overdueCount > 0"
          :value="overdueCount"
          label="Overdue"
          variant="dark"
          color="amber"
        />
      </template>
      <template #controls>
        <UserSelector id="todo-user" v-model="selectedUserId" :users="users" color="indigo" />
        <Button
          variant="gradient-indigo"
          size="sm"
          :disabled="!selectedUserId"
          class="group"
          @click="openCreateModal"
        >
          <span class="flex items-center gap-1.5">
            <Plus
              class="h-4 w-4 transition-transform duration-200 group-hover:rotate-90"
              :stroke-width="2.5"
              aria-hidden="true"
            />
            New Todo
          </span>
        </Button>
      </template>
      <template #actions>
        <TodoActionsBar
          v-model:view="view"
          v-model:active-filter="activeFilter"
          v-model:selected-priority="selectedPriority"
          v-model:due-date-filter="dueDateFilter"
          v-model:selected-tag="selectedTag"
          :filters="filters"
          :all-tags="allTags"
          :sort-field="sortField"
          :sort-dir="sortDir"
          @update:sort-field="setSort($event)"
          @toggle-dir="toggleSortDir()"
          @clear-all="clearAllFilters"
        />
      </template>
    </PageHeader>

    <SelectUserPrompt
      title="Select a user to manage their todos"
      :users-count="users.length"
      :selected-user-id="selectedUserId"
    />

    <LoadingSpinner v-if="isLoading" message="Loading todos..." color="indigo" />
    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadTodos" />

    <div v-else-if="selectedUserId">
      <div
        v-if="activeFilter === 'archived'"
        class="mb-4 flex items-center gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-500/20 dark:bg-amber-950/30 dark:text-amber-300"
        role="note"
      >
        <Archive class="h-5 w-5 shrink-0" :stroke-width="2" aria-hidden="true" />
        <span>
          <strong>Archived todos</strong> are hidden from your normal workspace. Restore one to make
          it active again.
        </span>
      </div>

      <TodoSearchBar v-model="searchQuery" :result-count="displayedTodos.length" />

      <EmptyState
        v-if="displayedTodos.length === 0"
        :title="emptyTitle"
        :description="emptyDescription"
        :icon="searchQuery ? 'search' : 'default'"
        color="indigo"
      />

      <template v-else>
        <TodoListView
          v-if="view === 'list'"
          v-model="listTodos"
          :expanded-ids="expandedTodoIds"
          @toggle="toggleTodo"
          @toggle-pin="togglePinned"
          @archive="archiveTodo"
          @restore="restoreTodo"
          @start-edit="openEditModal"
          @delete="confirmDelete"
          @view-screenshot="openScreenshot"
          @toggle-expand="toggleExpanded"
          @toggle-subtask="onSubtaskToggled"
          @row-mousedown="onRowMouseDown"
          @row-dragstart="onRowDragStart"
          @row-dragover="onRowDragOver"
          @row-drop="onRowDrop"
          @row-dragend="onRowDragEnd"
          @mobile-dragend="onDragEnd"
        />

        <TodoKanbanBoard
          v-else-if="view === 'kanban'"
          :todos="displayedTodos"
          @toggle="toggleTodo"
          @toggle-subtask="onSubtaskToggled"
          @toggle-pin="togglePinned"
          @archive="archiveTodo"
          @restore="restoreTodo"
          @view-screenshot="openScreenshot"
          @start-edit="openEditModal"
          @delete="confirmDelete"
          @reorder="onKanbanReorder"
          @priority-change="onKanbanPriorityChange"
        />

        <TodoGridView
          v-else-if="view === 'grid'"
          :todos="displayedTodos"
          @toggle="toggleTodo"
          @toggle-pin="togglePinned"
          @archive="archiveTodo"
          @restore="restoreTodo"
          @view-screenshot="openScreenshot"
          @start-edit="openEditModal"
          @delete="confirmDelete"
          @toggle-subtask="onSubtaskToggled"
        />
      </template>
    </div>

    <!-- Screenshot lightbox -->
    <Modal
      :open="screenshotModal.open"
      :title="screenshotModal.title"
      fullscreen
      @close="closeScreenshot"
    >
      <img
        :src="screenshotModal.url"
        alt="Todo screenshot"
        class="h-full w-full rounded-lg border border-gray-200 object-contain dark:border-slate-700"
      />
    </Modal>

    <TodoEditorModal
      :open="editorOpen"
      :editing-todo="editingTodo"
      :initial-due-date="editorDueDate"
      :user-id="selectedUserId ?? 0"
      @close="closeEditor"
      @submit="handleCreate"
      @update="handleUpdate"
      @delete="handleEditorDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { Archive, Plus } from '@lucide/vue';
import { useUser } from '../composables/useUser';
import { useTodoPage } from './todos/composables/useTodoPage';
import Button from './ui/Button.vue';
import UserSelector from './ui/UserSelector.vue';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import EmptyState from './ui/EmptyState.vue';
import SelectUserPrompt from './ui/SelectUserPrompt.vue';
import Modal from './ui/Modal.vue';
import TodoActionsBar from './todos/TodoActionsBar.vue';
import TodoSearchBar from './todos/TodoSearchBar.vue';
import TodoListView from './todos/views/TodoListView.vue';
import TodoKanbanBoard from './todos/views/TodoKanbanBoard.vue';
import TodoGridView from './todos/views/TodoGridView.vue';
import TodoEditorModal from './todos/editor/TodoEditorModal.vue';

const { users, currentUserId, setUser, clearUser } = useUser();
const selectedUserId = ref<number | null>(currentUserId.value);

const {
  todosApi: {
    todos,
    isLoading,
    error,
    activeFilter,
    searchQuery,
    filters,
    totalCount,
    activeCount,
    inProgressCount,
    completedCount,
    archivedCount,
    overdueCount,
    displayedTodos,
    loadTodos,
    toggleTodo,
    togglePinned,
    archiveTodo,
    restoreTodo,
    expandedTodoIds,
    toggleExpanded,
  },
  view,
  selectedPriority,
  dueDateFilter,
  allTags,
  selectedTag,
  sortField,
  sortDir,
  setSort,
  toggleSortDir,
  clearAllFilters,
  listTodos,
  onDragEnd,
  onRowMouseDown,
  onRowDragStart,
  onRowDragOver,
  onRowDrop,
  onRowDragEnd,
  onKanbanReorder,
  onKanbanPriorityChange,
  onSubtaskToggled,
  editorOpen,
  editingTodo,
  editorDueDate,
  openCreateModal,
  openEditModal,
  closeEditor,
  handleCreate,
  handleUpdate,
  handleEditorDelete,
  confirmDelete,
  screenshotModal,
  openScreenshot,
  closeScreenshot,
} = useTodoPage(selectedUserId);

watch(selectedUserId, (id) => {
  if (id) {
    setUser(id);
  } else {
    clearUser();
    todos.value = [];
  }
});

// Data load kicks in automatically via the immediate watcher inside useTodos.

const emptyTitle = computed(() => {
  if (searchQuery.value) return 'No matching todos';
  return activeFilter.value === 'archived' ? 'Archive is empty' : 'No todos here';
});

const emptyDescription = computed(() => {
  if (searchQuery.value) return `Nothing matches “${searchQuery.value}”. Try another search.`;
  return activeFilter.value === 'archived'
    ? 'Completed todos you archive will appear here.'
    : 'Create your first task above or change the filters.';
});
</script>
