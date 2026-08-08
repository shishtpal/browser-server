<template>
  <div class="mx-auto flex h-full max-w-full flex-col px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Schedule" title="Calendar" color="violet">
      <template #stats>
        <StatCard :value="stats.todayCount" label="Today" variant="dark" color="violet" />
        <StatCard :value="stats.overdueCount" label="Overdue" variant="primary" color="amber" />
        <StatCard :value="stats.completedCount" label="Done" variant="secondary" color="violet" />
      </template>
      <template #controls>
        <UserSelector id="calendar-user" v-model="selectedUserId" :users="users" color="violet" />
        <Button variant="gradient-violet" size="sm" @click="openCreateModal()">
          <span class="flex items-center gap-1.5">
            <Plus class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
            Add Todo
          </span>
        </Button>
      </template>
    </PageHeader>

    <SelectUserPrompt
      title="Select a user to view their calendar"
      :users-count="users.length"
      :selected-user-id="selectedUserId"
    />

    <LoadingSpinner v-if="isLoading" message="Loading calendar..." color="violet" />
    <ErrorBanner v-else-if="error" :message="error" :on-retry="loadTodos" />

    <div v-else-if="selectedUserId" class="flex min-h-0 flex-1 flex-col gap-3 sm:gap-4">
      <CalendarHeader
        :period-label="periodLabel"
        :current-view="view"
        @navigate="navigate"
        @today="goToToday"
        @change-view="(v) => (view = v)"
      />

      <div
        v-if="view === 'month'"
        class="flex-1 rounded-2xl border border-gray-200/80 bg-white/90 p-2 shadow-sm sm:p-3 dark:border-slate-700/80 dark:bg-slate-800/90"
      >
        <CalendarMonthView
          class="h-full"
          :days="days"
          @click="openCreateModal"
          @show-more="showDayTodos"
          @todo-click="openDetail"
          @todo-move="handleTodoMove"
        />
      </div>

      <div
        v-else-if="view === 'week'"
        class="rounded-2xl border border-gray-200/80 bg-white/90 p-2 shadow-sm sm:p-3 dark:border-slate-700/80 dark:bg-slate-800/90"
      >
        <CalendarWeekView
          :days="days"
          @click="openCreateModal"
          @todo-click="openDetail"
          @todo-move="handleTodoMove"
        />
      </div>

      <div
        v-else-if="view === 'day'"
        class="rounded-2xl border border-gray-200/80 bg-white/90 p-2 shadow-sm sm:p-3 dark:border-slate-700/80 dark:bg-slate-800/90"
      >
        <CalendarDayView :day="currentDayData" @todo-click="openDetail" />
      </div>

      <div
        v-else-if="view === 'year'"
        class="rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm sm:p-4 dark:border-slate-700/80 dark:bg-slate-800/90"
      >
        <CalendarYearView
          :year="currentDate.getFullYear()"
          :todos="todos"
          @month-click="jumpToMonth"
          @day-click="(date) => jumpToDate(date, 'day')"
          @year-change="jumpToYear"
        />
      </div>
    </div>

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

    <CalendarTodoDetail :todo="detailTodo" @close="closeDetail" @edit="editFromDetail" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { Plus } from '@lucide/vue';
import { useModal } from '@browser-server/shared-modal';
import { useUser } from '../composables/useUser';
import { useCalendarPage } from './calendar/composables/useCalendarPage';
import Button from './ui/Button.vue';
import UserSelector from './ui/UserSelector.vue';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import SelectUserPrompt from './ui/SelectUserPrompt.vue';
import CalendarHeader from './calendar/CalendarHeader.vue';
import CalendarMonthView from './calendar/CalendarMonthView.vue';
import CalendarWeekView from './calendar/CalendarWeekView.vue';
import CalendarDayView from './calendar/CalendarDayView.vue';
import CalendarYearView from './calendar/CalendarYearView.vue';
import CalendarTodoDetail from './calendar/CalendarTodoDetail.vue';
import TodoEditorModal from './todos/editor/TodoEditorModal.vue';

const { users, currentUserId, setUser, clearUser } = useUser();
const selectedUserId = ref<number | null>(currentUserId.value);

const {
  calendar: {
    currentDate,
    view,
    periodLabel,
    navigate,
    goToToday,
    jumpToDate,
    jumpToMonth,
    jumpToYear,
  },
  todosApi: { todos, isLoading, error, days, stats, loadTodos },
  editorOpen,
  editingTodo,
  editorDueDate,
  openCreateModal,
  closeEditor,
  handleCreate,
  handleUpdate,
  handleDelete,
  detailTodo,
  openDetail,
  closeDetail,
  editFromDetail,
  handleTodoMove,
  currentDayData,
} = useCalendarPage(selectedUserId);

watch(selectedUserId, (id) => {
  if (id) setUser(id);
  else clearUser();
});

/* ---------------------- delete confirmation from editor --------------------- */

const { confirmDelete: confirmDeleteModal } = useModal();

async function handleEditorDelete() {
  if (!editingTodo.value) return;
  const confirmed = await confirmDeleteModal(
    `Delete "${editingTodo.value.title}"?`,
    'This action cannot be undone.',
  );
  if (confirmed) await handleDelete();
}

function showDayTodos(date: string) {
  jumpToDate(date, 'day');
}
</script>
