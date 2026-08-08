<template>
  <!-- Filters toolbar -->
  <div
    class="flex w-full flex-wrap items-center gap-2 rounded-2xl border border-slate-200/80 bg-gradient-to-r from-slate-50/80 to-white/60 px-3 py-2.5 shadow-sm backdrop-blur-sm dark:border-slate-700/60 dark:from-slate-800/60 dark:to-slate-800/30"
  >
    <TodoViewToggle v-model:view="view" />

    <!-- Status tabs: scrollable on narrow screens -->
    <nav
      class="-mx-1 flex min-w-0 flex-1 scrollbar-none items-center gap-0.5 overflow-x-auto rounded-xl bg-slate-100/80 p-1 sm:flex-none dark:bg-slate-700/50"
      aria-label="Status filter"
    >
      <FilterPill
        v-for="f in filters"
        :key="f.value"
        :active="activeFilter === f.value"
        class="shrink-0"
        @click="activeFilter = f.value"
      >
        {{ f.label }}
      </FilterPill>
    </nav>

    <div class="hidden h-6 w-px bg-slate-200/80 sm:block dark:bg-slate-600/50" />

    <!-- Filter selects: 3-up grid on mobile, inline on desktop -->
    <div class="grid w-full grid-cols-3 gap-2 sm:flex sm:w-auto sm:flex-wrap sm:items-center">
      <TodoFilterSelect
        v-model="priorityProxy"
        label="Priority"
        :icon="Flag"
        :options="priorityOptions"
      />
      <TodoFilterSelect
        v-model="dueProxy"
        label="Due date"
        :icon="Calendar"
        :options="DUE_DATE_FILTERS"
      />
      <TodoFilterSelect v-model="selectedTag" label="Tags" :icon="Tag" :options="tagOptions" />

      <!-- Sort (list view only) -->
      <div v-if="view === 'list'" class="col-span-3 sm:col-span-1 sm:ml-auto">
        <TodoSortBar
          :sort-field="sortField"
          :sort-dir="sortDir"
          @update:sort-field="$emit('update:sortField', $event)"
          @toggle-dir="$emit('toggle-dir')"
        />
      </div>
    </div>
  </div>

  <!-- Active filter chips -->
  <Transition
    enter-active-class="transition-all duration-200 ease-out"
    enter-from-class="-translate-y-1 opacity-0"
    enter-to-class="translate-y-0 opacity-100"
    leave-active-class="transition-all duration-150 ease-in"
    leave-from-class="translate-y-0 opacity-100"
    leave-to-class="-translate-y-1 opacity-0"
  >
    <div v-if="activeChips.length" class="flex flex-wrap items-center gap-2">
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
          v-for="chip in activeChips"
          :key="chip.key"
          :active="true"
          variant="tag"
          class="py-1"
          @click="chip.clear()"
        >
          <span class="flex items-center gap-1 text-[10px]">
            <component :is="chip.icon" class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
            {{ chip.label }}
            <X
              class="h-3 w-3 opacity-60 transition-opacity hover:opacity-100"
              :stroke-width="2.5"
              aria-hidden="true"
            />
          </span>
        </FilterPill>
      </TransitionGroup>

      <button
        type="button"
        class="ml-1 rounded-lg px-2 py-0.5 text-[11px] font-medium text-slate-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:text-slate-500 dark:hover:bg-red-500/10 dark:hover:text-red-400"
        @click="$emit('clear-all')"
      >
        Clear all
      </button>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import type { DueDateFilter, TodoFilter, TodoPriority, TodoSortField, TodoView } from '../../types';
import { computed } from 'vue';
import { Calendar, Flag, Tag, X, type LucideIcon } from '@lucide/vue';
import FilterPill from '../ui/FilterPill.vue';
import TodoViewToggle from './TodoViewToggle.vue';
import TodoSortBar from './TodoSortBar.vue';
import TodoFilterSelect from './TodoFilterSelect.vue';
import {
  DUE_DATE_FILTERS,
  DUE_DATE_FILTER_LABELS,
  PRIORITY_META,
  PRIORITY_ORDER,
} from './todoFormat';

const props = defineProps<{
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
  'update:sortField': [value: TodoSortField];
  'toggle-dir': [];
  'clear-all': [];
}>();

const priorityOptions = PRIORITY_ORDER.map((value) => ({
  value,
  label: PRIORITY_META[value].label,
}));
const tagOptions = computed(() => props.allTags.map((t) => ({ value: t, label: t })));

/** TodoFilterSelect binds `string | null`; widen/narrow the typed models. */
const priorityProxy = computed<string | null>({
  get: () => selectedPriority.value,
  set: (v) => (selectedPriority.value = v as TodoPriority | null),
});
const dueProxy = computed<string | null>({
  get: () => dueDateFilter.value,
  set: (v) => (dueDateFilter.value = v as DueDateFilter),
});

const activeChips = computed(() => {
  const chips: { key: string; label: string; icon: LucideIcon; clear: () => void }[] = [];
  if (selectedPriority.value) {
    chips.push({
      key: 'priority',
      label: PRIORITY_META[selectedPriority.value]?.label ?? selectedPriority.value,
      icon: Flag,
      clear: () => (selectedPriority.value = null),
    });
  }
  if (dueDateFilter.value) {
    chips.push({
      key: 'due',
      label: DUE_DATE_FILTER_LABELS[dueDateFilter.value] ?? dueDateFilter.value,
      icon: Calendar,
      clear: () => (dueDateFilter.value = null),
    });
  }
  if (selectedTag.value) {
    chips.push({
      key: 'tag',
      label: selectedTag.value,
      icon: Tag,
      clear: () => (selectedTag.value = null),
    });
  }
  return chips;
});
</script>

<style scoped>
.scrollbar-none {
  scrollbar-width: none;
}
.scrollbar-none::-webkit-scrollbar {
  display: none;
}
</style>
