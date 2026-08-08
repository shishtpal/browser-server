<template>
  <!-- Filters Toolbar -->
  <div
    class="flex w-full flex-wrap items-center gap-2.5 rounded-2xl border border-slate-200/80 bg-gradient-to-r from-slate-50/80 to-white/60 px-3 py-2.5 shadow-sm backdrop-blur-sm dark:border-slate-700/60 dark:from-slate-800/60 dark:to-slate-800/30"
  >
    <TodoViewToggle :view="view" @update:view="view = $event" />

    <!-- Status Tabs -->
    <nav class="flex items-center gap-0.5 rounded-xl bg-slate-100/80 p-1 dark:bg-slate-700/50">
      <FilterPill
        v-for="f in filters"
        :key="f.value"
        :active="activeFilter === f.value"
        @click="activeFilter = f.value"
      >
        {{ f.label }}
      </FilterPill>
    </nav>

    <!-- Divider -->
    <div class="hidden h-6 w-px bg-slate-200/80 sm:block dark:bg-slate-600/50" />

    <!-- Filter Selects -->
    <div class="flex flex-wrap items-center gap-2">
      <!-- Priority -->
      <div class="group relative">
        <svg
          class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-slate-400 transition-colors group-focus-within:text-indigo-500"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
          <line x1="4" x2="4" y1="22" y2="15" />
        </svg>
        <select
          v-model="selectedPriority"
          class="cursor-pointer appearance-none rounded-xl border border-slate-200/80 bg-white/80 py-1.5 pr-7 pl-8 text-xs font-semibold text-slate-600 shadow-sm transition-all hover:border-indigo-300 hover:shadow focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none dark:border-slate-600/60 dark:bg-slate-700/60 dark:text-slate-300 dark:hover:border-indigo-500/50 dark:focus:border-indigo-500"
        >
          <option :value="null">Priority</option>
          <option v-for="p in allPriorityOptions" :key="p.value" :value="p.value">
            {{ p.label }}
          </option>
        </select>
        <svg
          class="pointer-events-none absolute top-1/2 right-2 h-3 w-3 -translate-y-1/2 text-slate-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </div>

      <!-- Due Date -->
      <div class="group relative">
        <svg
          class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-slate-400 transition-colors group-focus-within:text-indigo-500"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <rect width="18" height="18" x="3" y="4" rx="2" />
          <line x1="16" x2="16" y1="2" y2="6" />
          <line x1="8" x2="8" y1="2" y2="6" />
          <line x1="3" x2="21" y1="10" y2="10" />
        </svg>
        <select
          v-model="dueDateFilter"
          class="cursor-pointer appearance-none rounded-xl border border-slate-200/80 bg-white/80 py-1.5 pr-7 pl-8 text-xs font-semibold text-slate-600 shadow-sm transition-all hover:border-indigo-300 hover:shadow focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none dark:border-slate-600/60 dark:bg-slate-700/60 dark:text-slate-300 dark:hover:border-indigo-500/50 dark:focus:border-indigo-500"
        >
          <option :value="null">Due date</option>
          <option value="overdue">Overdue</option>
          <option value="today">Due today</option>
          <option value="this_week">This week</option>
        </select>
        <svg
          class="pointer-events-none absolute top-1/2 right-2 h-3 w-3 -translate-y-1/2 text-slate-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </div>

      <!-- Tags -->
      <div class="group relative">
        <svg
          class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-slate-400 transition-colors group-focus-within:text-indigo-500"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path
            d="M12 2H2v10l9.29 9.29c.94.94 2.48.94 3.42 0l6.58-6.58c.94-.94.94-2.48 0-3.42L12 2Z"
          />
          <path d="M7 7h.01" />
        </svg>
        <select
          v-model="selectedTag"
          class="cursor-pointer appearance-none rounded-xl border border-slate-200/80 bg-white/80 py-1.5 pr-7 pl-8 text-xs font-semibold text-slate-600 shadow-sm transition-all hover:border-indigo-300 hover:shadow focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none dark:border-slate-600/60 dark:bg-slate-700/60 dark:text-slate-300 dark:hover:border-indigo-500/50 dark:focus:border-indigo-500"
        >
          <option :value="null">Tags</option>
          <option v-for="t in allTags" :key="t" :value="t">{{ t }}</option>
        </select>
        <svg
          class="pointer-events-none absolute top-1/2 right-2 h-3 w-3 -translate-y-1/2 text-slate-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </div>
    </div>

    <!-- Sort (pushed right) -->
    <div v-if="view === 'list'" class="ml-auto">
      <TodoSortBar
        :sort-field="sortField"
        :sort-dir="sortDir"
        @update:sort-field="$emit('update:sortField', $event)"
        @toggle-dir="$emit('toggle-dir')"
      />
    </div>
  </div>

  <!-- Active Filter Chips -->
  <Transition
    enter-active-class="transition-all duration-200 ease-out"
    enter-from-class="-translate-y-1 opacity-0"
    enter-to-class="translate-y-0 opacity-100"
    leave-active-class="transition-all duration-150 ease-in"
    leave-from-class="translate-y-0 opacity-100"
    leave-to-class="-translate-y-1 opacity-0"
  >
    <div
      v-if="selectedPriority || dueDateFilter || selectedTag"
      class="flex flex-wrap items-center gap-2"
    >
      <span
        class="text-[11px] font-medium tracking-wide text-slate-400 uppercase dark:text-slate-500"
      >
        Filters
      </span>

      <TransitionGroup
        enter-active-class="transition-all duration-200 ease-out"
        enter-from-class="scale-90 opacity-0"
        enter-to-class="scale-100 opacity-100"
        leave-active-class="transition-all duration-150 ease-in"
        leave-from-class="scale-100 opacity-100"
        leave-to-class="scale-90 opacity-0"
      >
        <FilterPill
          v-if="selectedPriority"
          key="priority"
          :active="true"
          variant="tag"
          @click="selectedPriority = null"
        >
          <span class="flex items-center gap-1">
            <svg
              class="h-3 w-3"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
              <line x1="4" x2="4" y1="22" y2="15" />
            </svg>
            {{ selectedPriority }}
            <svg
              class="h-3 w-3 opacity-60 transition-opacity hover:opacity-100"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </span>
        </FilterPill>

        <FilterPill
          v-if="dueDateFilter"
          key="dueDate"
          :active="true"
          variant="tag"
          @click="dueDateFilter = null"
        >
          <span class="flex items-center gap-1">
            <svg
              class="h-3 w-3"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <rect width="18" height="18" x="3" y="4" rx="2" />
              <line x1="16" x2="16" y1="2" y2="6" />
              <line x1="8" x2="8" y1="2" y2="6" />
              <line x1="3" x2="21" y1="10" y2="10" />
            </svg>
            {{ dueDateLabel }}
            <svg
              class="h-3 w-3 opacity-60 transition-opacity hover:opacity-100"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </span>
        </FilterPill>

        <FilterPill
          v-if="selectedTag"
          key="tag"
          :active="true"
          variant="tag"
          @click="selectedTag = null"
        >
          <span class="flex items-center gap-1">
            <svg
              class="h-3 w-3"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path
                d="M12 2H2v10l9.29 9.29c.94.94 2.48.94 3.42 0l6.58-6.58c.94-.94.94-2.48 0-3.42L12 2Z"
              />
              <path d="M7 7h.01" />
            </svg>
            {{ selectedTag }}
            <svg
              class="h-3 w-3 opacity-60 transition-opacity hover:opacity-100"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              stroke-width="2.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </span>
        </FilterPill>
      </TransitionGroup>

      <button
        class="ml-1 rounded-lg px-2 py-0.5 text-[11px] font-medium text-slate-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:text-slate-500 dark:hover:bg-red-500/10 dark:hover:text-red-400"
        @click="$emit('clear-all')"
      >
        Clear all
      </button>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import type { TodoView, TodoPriority, TodoSortField, TodoFilter, DueDateFilter } from '../../types';
import { computed } from 'vue';
import FilterPill from '../ui/FilterPill.vue';
import TodoViewToggle from './TodoViewToggle.vue';
import TodoSortBar from './TodoSortBar.vue';

const allPriorityOptions: { value: TodoPriority; label: string }[] = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
  { value: 'urgent', label: 'Urgent' },
];

defineProps<{
  filters: { label: string; value: TodoFilter }[];
  allTags: string[];
  sortField: TodoSortField;
  sortDir: 'asc' | 'desc';
}>();

const view = defineModel<TodoView>('view', { required: true });
const activeFilter = defineModel<TodoFilter>('activeFilter', { required: true });
const selectedPriority = defineModel<TodoPriority | null>('selectedPriority', { required: true });
const dueDateFilter = defineModel<DueDateFilter>('dueDateFilter', { required: true });
const selectedTag = defineModel<string | null>('selectedTag', { required: true });

defineEmits<{
  'new-todo': [];
  'update:sortField': [value: TodoSortField];
  'toggle-dir': [];
  'clear-all': [];
}>();

const dueDateLabel = computed(() => {
  if (!dueDateFilter.value) return '';
  const labels: Record<string, string> = {
    overdue: 'Overdue',
    today: 'Today',
    this_week: 'This week',
  };
  return labels[dueDateFilter.value] || '';
});
</script>
