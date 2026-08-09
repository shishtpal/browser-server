<template>
  <form
    class="mb-4 rounded-2xl border border-gray-200 bg-white p-4 shadow-sm transition-colors dark:border-slate-700 dark:bg-slate-800/90"
    aria-label="Queue a background task"
    @submit.prevent="handleSubmit"
  >
    <div class="mb-2 flex items-center gap-2">
      <span
        class="grid h-7 w-7 place-items-center rounded-lg bg-violet-100 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400"
      >
        <Bot class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
      </span>
      <label
        for="task-prompt"
        class="text-xs font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
      >
        New task
      </label>
    </div>

    <textarea
      id="task-prompt"
      v-model="draft"
      rows="3"
      :disabled="isSubmitting"
      placeholder="Describe what the agent should do. It runs unattended, so avoid anything that needs your input."
      class="w-full resize-y rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-slate-800 transition outline-none focus:border-violet-400 focus:ring-2 focus:ring-violet-100 disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:focus:ring-violet-900/30"
      @keydown.ctrl.enter.prevent="handleSubmit"
      @keydown.meta.enter.prevent="handleSubmit"
    />

    <div class="mt-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <p class="text-[11px] text-slate-500 dark:text-slate-400">
        <span class="hidden sm:inline">Ctrl+Enter to submit · </span>{{ workersLabel(workers) }}
      </p>
      <Button
        type="submit"
        variant="gradient-violet"
        size="sm"
        class="w-full sm:w-auto"
        :loading="isSubmitting"
        loading-text="Queueing..."
        :disabled="!draft.trim()"
      >
        <span class="inline-flex items-center gap-1.5">
          <Send class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Queue task
        </span>
      </Button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { Bot, Send } from '@lucide/vue';
import Button from '../ui/Button.vue';
import { workersLabel } from './taskFormat';

defineProps<{
  workers: number;
  isSubmitting: boolean;
}>();

const emit = defineEmits<{ submit: [prompt: string] }>();

const draft = ref('');

function handleSubmit() {
  emit('submit', draft.value);
  draft.value = '';
}
</script>
