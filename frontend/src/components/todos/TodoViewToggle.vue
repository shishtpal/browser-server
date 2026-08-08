<template>
  <div
    class="flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-slate-700 dark:bg-slate-800/80"
    role="group"
    aria-label="Todo view"
  >
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      :aria-pressed="view === option.value"
      :title="`${option.label} view`"
      :class="[
        'flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-[11px] font-black transition sm:px-3',
        view === option.value
          ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-700 dark:text-white'
          : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200',
      ]"
      @click="$emit('update:view', option.value)"
    >
      <component :is="option.icon" class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
      <span class="hidden sm:inline">{{ option.label }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import type { TodoView } from '../../types';
import { LayoutGrid, Columns3, List, type LucideIcon } from '@lucide/vue';

defineProps<{ view: TodoView }>();
defineEmits<{ 'update:view': [value: TodoView] }>();

const options: { value: TodoView; label: string; icon: LucideIcon }[] = [
  { value: 'list', label: 'List', icon: List },
  { value: 'kanban', label: 'Kanban', icon: Columns3 },
  { value: 'grid', label: 'Grid', icon: LayoutGrid },
];
</script>
