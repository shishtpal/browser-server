<template>
  <div class="flex items-center gap-0.5">
    <button
      type="button"
      :title="todo.pinned ? 'Unpin' : 'Pin to top'"
      :aria-label="todo.pinned ? 'Unpin todo' : 'Pin todo'"
      :aria-pressed="todo.pinned"
      class="grid h-8 w-8 place-items-center rounded-lg transition"
      :class="
        todo.pinned
          ? 'bg-indigo-50 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300'
          : 'text-slate-400 hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400'
      "
      @click="$emit('toggle-pin', todo)"
    >
      <Pin v-if="!todo.pinned" class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
      <PinOff v-else class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
    </button>

    <button
      v-if="todo.status === 'archived'"
      type="button"
      title="Restore todo"
      aria-label="Restore todo"
      class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-400"
      @click="$emit('restore', todo)"
    >
      <ArchiveRestore class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
    </button>
    <button
      v-else-if="todo.status === 'completed'"
      type="button"
      title="Archive todo"
      aria-label="Archive todo"
      class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-amber-50 hover:text-amber-600 dark:hover:bg-amber-900/20 dark:hover:text-amber-400"
      @click="$emit('archive', todo)"
    >
      <Archive class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
    </button>

    <button
      v-if="todo.status !== 'archived'"
      type="button"
      title="Edit todo"
      aria-label="Edit todo"
      class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-indigo-50 hover:text-indigo-600 dark:hover:bg-indigo-500/10 dark:hover:text-indigo-400"
      @click="$emit('start-edit', todo)"
    >
      <Pencil class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
    </button>

    <button
      type="button"
      title="Delete todo"
      aria-label="Delete todo"
      class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20 dark:hover:text-red-400"
      @click="$emit('delete', todo.id)"
    >
      <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Archive, ArchiveRestore, Pencil, Pin, PinOff, Trash2 } from '@lucide/vue';
import type { Todo } from '../../types';

defineProps<{ todo: Todo }>();

defineEmits<{
  'toggle-pin': [todo: Todo];
  archive: [todo: Todo];
  restore: [todo: Todo];
  'start-edit': [todo: Todo];
  delete: [id: number];
}>();
</script>
