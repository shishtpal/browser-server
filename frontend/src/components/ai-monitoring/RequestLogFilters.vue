<template>
  <form
    class="grid gap-3 rounded-2xl border border-slate-200 bg-white p-3 shadow-sm sm:grid-cols-2 sm:p-4 lg:grid-cols-5 dark:border-white/10 dark:bg-slate-900"
    @submit.prevent="$emit('apply')"
  >
    <label class="block text-xs font-bold text-slate-600 dark:text-slate-300">
      Source
      <select
        :value="source"
        :class="controlClass"
        @change="
          $emit('update:source', ($event.target as HTMLSelectElement).value as SourceValue);
          $emit('apply');
        "
      >
        <option value="">All sources</option>
        <option value="chat">Chat</option>
        <option value="task_agent">Task agent</option>
      </select>
    </label>

    <label class="block text-xs font-bold text-slate-600 dark:text-slate-300">
      Status
      <select
        :value="status"
        :class="controlClass"
        @change="
          $emit('update:status', ($event.target as HTMLSelectElement).value as StatusValue);
          $emit('apply');
        "
      >
        <option value="">All statuses</option>
        <option value="success">Success</option>
        <option value="error">Error</option>
        <option value="cancelled">Cancelled</option>
      </select>
    </label>

    <label class="block text-xs font-bold text-slate-600 dark:text-slate-300">
      Conversation ID
      <input
        :value="conversationInput"
        :class="controlClass"
        placeholder="Optional ID"
        @input="$emit('update:conversationInput', ($event.target as HTMLInputElement).value)"
      />
    </label>

    <label class="block text-xs font-bold text-slate-600 dark:text-slate-300">
      Task ID
      <input
        :value="taskInput"
        :class="controlClass"
        placeholder="Optional ID"
        @input="$emit('update:taskInput', ($event.target as HTMLInputElement).value)"
      />
    </label>

    <div class="flex items-end gap-2">
      <Button type="submit" size="sm" class="flex-1 sm:flex-none">
        <span class="inline-flex items-center gap-1.5">
          <ListFilter class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Apply
        </span>
      </Button>
      <Button type="button" variant="ghost" size="sm" @click="$emit('clear')">Clear</Button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ListFilter } from '@lucide/vue';
import Button from '../ui/Button.vue';

export type SourceValue = '' | 'chat' | 'task_agent';
export type StatusValue = '' | 'success' | 'error' | 'cancelled';

defineProps<{
  source: SourceValue;
  status: StatusValue;
  conversationInput: string;
  taskInput: string;
}>();

defineEmits<{
  'update:source': [value: SourceValue];
  'update:status': [value: StatusValue];
  'update:conversationInput': [value: string];
  'update:taskInput': [value: string];
  apply: [];
  clear: [];
}>();

const controlClass =
  'mt-1 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm font-normal text-slate-900 transition focus:border-cyan-500 focus:ring-4 focus:ring-cyan-500/20 focus:outline-none dark:border-white/10 dark:bg-slate-800 dark:text-white';
</script>
