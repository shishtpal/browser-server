<template>
  <div
    class="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-900/40 dark:bg-amber-950/20"
    role="alert"
  >
    <p class="mb-2 text-[0.85em] text-amber-800 dark:text-amber-200">
      {{ hint }}
    </p>
    <div class="flex flex-wrap items-center gap-2">
      <button
        class="inline-flex items-center gap-1.5 rounded-md bg-emerald-600 px-3 py-1.5 text-[0.85em] font-semibold text-white transition hover:bg-emerald-700"
        type="button"
        @click="$emit('decision', true, '')"
      >
        <CircleCheck class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        {{ isRetry ? 'Resume' : 'Allow' }}
      </button>
      <button
        class="inline-flex items-center gap-1.5 rounded-md border border-red-200 bg-white px-3 py-1.5 text-[0.85em] font-semibold text-red-600 transition hover:bg-red-50 dark:border-red-900/60 dark:bg-slate-950 dark:hover:bg-red-950/30"
        type="button"
        @click="$emit('decision', false, '')"
      >
        <OctagonX class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        {{ isRetry ? 'Stop' : 'Reject' }}
      </button>
      <input
        v-model="commentDraft"
        class="min-w-40 flex-1 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-[0.85em] outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
        :placeholder="
          isRetry ? 'Or add instructions before resuming…' : 'Or send feedback instead…'
        "
        @keydown.enter.prevent="submitComment"
      />
      <button
        class="inline-flex items-center gap-1 rounded-md bg-slate-700 px-3 py-1.5 text-[0.85em] font-semibold text-white transition hover:bg-slate-800 disabled:opacity-40 dark:bg-slate-200 dark:text-slate-900 dark:hover:bg-white"
        type="button"
        :disabled="!commentDraft.trim()"
        @click="submitComment"
      >
        Send
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { CircleCheck, OctagonX } from '@lucide/vue';

const props = defineProps<{
  isRetry: boolean;
  retryHint?: string;
}>();

const emit = defineEmits<{
  decision: [approved: boolean, comment: string];
}>();

const commentDraft = ref('');

const hint = computed(() => {
  if (props.isRetry) {
    return (
      props.retryHint ||
      'The AI provider failed while continuing after a tool call. Resume without the last tool-call turn?'
    );
  }
  return 'Review the command or arguments before allowing this tool.';
});

function submitComment() {
  const text = commentDraft.value.trim();
  if (!text) return;
  emit('decision', false, text);
  commentDraft.value = '';
}
</script>
