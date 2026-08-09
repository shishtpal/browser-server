<template>
  <form
    class="mb-4 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-white/10 dark:bg-slate-800/90"
    aria-label="Add a history entry"
    @submit.prevent="onSubmit"
  >
    <div class="grid gap-2 sm:grid-cols-[1fr_1fr_7rem_auto] sm:items-center">
      <div class="relative">
        <Link2
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="url"
          type="url"
          placeholder="https://example.com"
          required
          class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:ring-4 focus:ring-violet-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30"
        />
      </div>

      <div class="relative">
        <Type
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="title"
          type="text"
          placeholder="Page title"
          required
          class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:ring-4 focus:ring-violet-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30"
        />
      </div>

      <div class="relative">
        <Clock
          class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
          aria-hidden="true"
        />
        <input
          v-model="durationStr"
          type="number"
          min="0"
          placeholder="Duration (s)"
          class="w-full rounded-lg border border-gray-300 bg-gray-50 py-2 pr-3 pl-9 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-violet-400 focus:ring-4 focus:ring-violet-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-violet-900/30"
        />
      </div>

      <Button type="submit" variant="gradient-violet" size="sm">
        <span class="inline-flex items-center gap-1.5">
          <Plus class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
          Add
        </span>
      </Button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { Clock, Link2, Plus, Type } from '@lucide/vue';
import Button from '../ui/Button.vue';
import type { HistoryCreateInput } from './composables/useHistory';

const emit = defineEmits<{ submit: [input: HistoryCreateInput] }>();

const url = ref('');
const title = ref('');
const durationStr = ref('');

const onSubmit = () => {
  emit('submit', {
    url: url.value,
    title: title.value,
    duration: durationStr.value ? Number(durationStr.value) : undefined,
  });
  url.value = '';
  title.value = '';
  durationStr.value = '';
};
</script>
