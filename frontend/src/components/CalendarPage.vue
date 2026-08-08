<template>
  <div class="mx-auto flex h-full max-w-full flex-col px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Schedule" title="Calendar" color="violet">
      <template #stats>
        <StatCard :value="todosStats.todayCount" label="Today" variant="dark" color="violet" />
        <StatCard
          :value="todosStats.overdueCount"
          label="Overdue"
          variant="primary"
          color="amber"
        />
        <StatCard
          :value="todosStats.completedCount"
          label="Done"
          variant="secondary"
          color="violet"
        />
      </template>
      <template #controls>
        <UserSelector id="calendar-user" v-model="selectedUserId" :users="users" color="violet" />
        <Button variant="gradient-violet" size="sm" @click="openCreateModal()">
          <span class="flex items-center gap-1">
            <svg
              class="h-3.5 w-3.5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M12 5v14M5 12h14" />
            </svg>
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

    <div v-else-if="selectedUserId" class="flex min-h-0 flex-1 flex-col gap-4">
      <CalendarHeader
        :period-label="periodLabel"
        :current-view="view"
        @navigate="navigate"
        @today="goToToday"
        @change-view="(v) => (view = v)"
      />

      <div
        v-if="view === 'month'"
        class="flex-1 rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm dark:border-slate-700/80 dark:bg-slate-800/90"
      >
        <CalendarMonthView
          class="h-full"
          :days="days"
          @click="openCreateModal"
          @show-more="showDayTodos"
          @todo-click="openEditModal"
          @todo-move="handleTodoMove"
        />
      </div>

      <div
        v-else-if="view === 'week'"
        class="rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm dark:border-slate-700/80 dark:bg-slate-800/90"
      >
        <CalendarWeekView
          :days="weekDays"
          @click="openCreateModal"
          @todo-click="openEditModal"
          @todo-move="handleTodoMove"
        />
      </div>

      <div
        v-else-if="view === 'day'"
        class="rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm dark:border-slate-700/80 dark:bg-slate-800/90"
      >
        <CalendarDayView :day="currentDayData" @todo-click="openEditModal" />
      </div>

      <div
        v-else-if="view === 'year'"
        class="rounded-2xl border border-gray-200/80 bg-white/90 p-4 shadow-sm dark:border-slate-700/80 dark:bg-slate-800/90"
      >
        <CalendarYearView
          :year="currentDate.getFullYear()"
          :todos="todos"
          @month-click="onMonthClick"
          @day-click="onYearDayClick"
          @year-change="onYearChange"
        />
      </div>
    </div>

    <CalendarTodoModal
      :open="modalOpen"
      :editing-todo="editingTodo"
      :initial-due-date="modalDueDate"
      :user-id="selectedUserId!"
      @close="closeModal"
      @submit="handleCreate"
      @update="handleUpdate"
      @delete="handleDelete"
    />

    <CalendarTodoDetail :todo="detailTodo" @close="closeDetail" @edit="editFromDetail" />
  </div>
</template>

<script setup lang="ts">
import type { Todo, CreateTodoInput } from '../types';
import { ref, watch, computed } from 'vue';
import { format } from 'date-fns';
import { useUser } from '../composables/useUser';
import { useCalendar } from '../composables/useCalendar';
import { useCalendarTodos } from '../composables/useCalendarTodos';
import PageHeader from './ui/PageHeader.vue';
import UserSelector from './ui/UserSelector.vue';
import Button from './ui/Button.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import SelectUserPrompt from './ui/SelectUserPrompt.vue';
import CalendarHeader from './calendar/CalendarHeader.vue';
import CalendarMonthView from './calendar/CalendarMonthView.vue';
import CalendarWeekView from './calendar/CalendarWeekView.vue';
import CalendarDayView from './calendar/CalendarDayView.vue';
import CalendarYearView from './calendar/CalendarYearView.vue';
import StatCard from './ui/StatCard.vue';
import CalendarTodoModal from './calendar/CalendarTodoModal.vue';
import CalendarTodoDetail from './calendar/CalendarTodoDetail.vue';

const { users, currentUserId, setUser, clearUser } = useUser();
const selectedUserId = ref<number | null>(currentUserId.value);

const { currentDate, view, dateRange, periodLabel, navigate, goToToday } = useCalendar();
const { todos, isLoading, error, days, stats, loadTodos, addTodo, updateTodoItem, removeTodo } =
  useCalendarTodos(selectedUserId, dateRange);

const todosStats = computed(() => stats.value);
const modalOpen = ref(false);
const editingTodo = ref<Todo | null>(null);
const modalDueDate = ref('');
const detailTodo = ref<Todo | null>(null);

// For week view, only pass the 7 days of the current week
const weekDays = computed(() => {
  if (view.value !== 'week') return [];
  return days.value;
});

// For day view, find the current date's data
const currentDayData = computed(() => {
  if (view.value !== 'day') return undefined;
  const dateStr = format(currentDate.value, 'yyyy-MM-dd');
  return days.value.find((d) => d.date === dateStr) ?? days.value[0];
});

watch(selectedUserId, (id) => {
  if (id) {
    setUser(id);
    loadTodos();
  } else {
    clearUser();
    todos.value = [];
  }
});

if (selectedUserId.value) {
  setUser(selectedUserId.value);
  loadTodos();
}

function openCreateModal(date?: string) {
  editingTodo.value = null;
  modalDueDate.value = date || format(new Date(), 'yyyy-MM-dd');
  modalOpen.value = true;
}

function openEditModal(todo: Todo) {
  detailTodo.value = todo;
}

function closeDetail() {
  detailTodo.value = null;
}

function editFromDetail(todo: Todo) {
  detailTodo.value = null;
  editingTodo.value = todo;
  modalDueDate.value = todo.start_date || '';
  modalOpen.value = true;
}

function closeModal() {
  modalOpen.value = false;
  editingTodo.value = null;
}

async function handleCreate(data: CreateTodoInput) {
  await addTodo(data);
}

async function handleUpdate(id: number, data: Partial<Todo>) {
  await updateTodoItem(id, data);
}

async function handleTodoMove(payload: { todo: Todo; date: string }) {
  await updateTodoItem(payload.todo.id, { start_date: payload.date });
}

async function handleDelete() {
  if (!editingTodo.value) return;
  await removeTodo(editingTodo.value.id);
  closeModal();
}

function showDayTodos(date: string) {
  // Navigate to day view for that date
  currentDate.value = new Date(date + 'T00:00:00');
  view.value = 'day';
}

function onMonthClick(month: number) {
  currentDate.value = new Date(currentDate.value.getFullYear(), month, 1);
  view.value = 'month';
}

function onYearDayClick(date: string) {
  currentDate.value = new Date(date + 'T00:00:00');
  view.value = 'day';
}

function onYearChange(year: number) {
  currentDate.value = new Date(year, currentDate.value.getMonth(), 1);
}
</script>
