<template>
  <article
    class="group relative w-full max-w-[96%] pr-8 text-slate-700 sm:max-w-[92%] dark:text-slate-300"
  >
    <button
      class="absolute top-0 right-0 hidden rounded-md p-1.5 text-slate-400 transition group-focus-within:block group-hover:block hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-500/10 dark:hover:text-red-400"
      title="Delete message"
      aria-label="Delete message"
      type="button"
      @click="$emit('delete', message.id)"
    >
      <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
    </button>

    <!-- Header toggle -->
    <button
      class="group/toggle flex w-full items-center gap-1.5 rounded-md py-1 text-left text-[0.8em]"
      type="button"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <Terminal
        class="h-3.5 w-3.5 shrink-0 text-slate-400 dark:text-slate-500"
        aria-hidden="true"
      />
      <span class="text-slate-400 dark:text-slate-500">{{ isRetry ? 'requested' : 'used' }}</span>
      <span class="min-w-0 truncate font-mono font-medium text-slate-700 dark:text-slate-200">{{
        label
      }}</span>
      <span
        class="ml-0.5 inline-flex shrink-0 items-center gap-1 font-medium"
        :class="status.className"
      >
        <component :is="statusIcon" class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        {{ status.label }}
      </span>
      <ChevronDown
        class="ml-0.5 h-3.5 w-3.5 shrink-0 text-slate-400 transition-transform group-hover/toggle:text-slate-600 dark:group-hover/toggle:text-slate-200"
        :class="{ 'rotate-180': !expanded }"
        aria-hidden="true"
      />
    </button>

    <div v-show="expanded" class="mt-2 space-y-2">
      <!-- Question call: answer form -->
      <ChatQuestionForm
        v-if="questionRequest && message.status === 'pending' && !toolData.decision"
        :context="questionRequest.context"
        :questions="questionRequest.questions"
        @submit="submitAnswers"
      />

      <!-- Approval/retry flow -->
      <ToolApprovalCard
        v-else-if="message.status === 'pending' && !toolData.decision"
        :is-retry="isRetry"
        :retry-hint="retryHint"
        @decision="(approved, comment) => emitDecision(approved, comment)"
      />

      <!-- Comment banner -->
      <div
        v-if="toolData.decision === 'commented' && feedbackComment"
        class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-[0.85em] text-amber-800 dark:border-amber-800/40 dark:bg-amber-900/20 dark:text-amber-200"
      >
        <span class="font-semibold">Your feedback:</span> {{ feedbackComment }}
      </div>

      <!-- Sections -->
      <section
        v-for="section in sections"
        :key="section.label"
        class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-white/10 dark:bg-slate-900"
      >
        <header
          class="flex h-7 items-center justify-between border-b border-slate-200 bg-slate-50 px-2.5 dark:border-white/10 dark:bg-slate-800/60"
        >
          <span
            class="text-[0.65em] font-semibold tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            {{ section.label }}
          </span>
          <button
            class="grid h-6 w-6 place-items-center rounded text-slate-400 transition hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-slate-200"
            type="button"
            :title="`Copy ${section.label.toLowerCase()}`"
            :aria-label="`Copy ${section.label.toLowerCase()}`"
            @click="$emit('copy', section.copyValue)"
          >
            <Copy class="h-3.5 w-3.5" :stroke-width="2" aria-hidden="true" />
          </button>
        </header>
        <pre
          class="max-h-64 overflow-auto px-2.5 py-2 font-mono text-[0.78em] leading-[1.55] break-words whitespace-pre-wrap text-slate-800 dark:text-slate-200"
          >{{ section.content }}</pre>
      </section>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import {
  ChevronDown,
  CircleCheck,
  CircleAlert,
  LoaderCircle,
  Copy,
  Terminal,
  Trash2,
  type LucideIcon,
} from '@lucide/vue';
import type { AIMessage } from '@browser-server/shared-types';
import {
  deriveToolData,
  isRecord,
  parseQuestionRequest,
  retryMessage,
  toolLabel,
  toolSections,
  toolStatus,
} from './messageTools';
import ChatQuestionForm from '../ChatQuestionForm.vue';
import ToolApprovalCard from './ToolApprovalCard.vue';

const props = defineProps<{ message: AIMessage }>();

const emit = defineEmits<{
  copy: [content: string];
  delete: [messageId: string];
  'tool-decision': [callId: string, approved: boolean, comment: string];
}>();

const expanded = ref(true);

const toolData = computed(() => deriveToolData(props.message));
const label = computed(() => toolLabel(toolData.value.name));
const status = computed(() => toolStatus(props.message, toolData.value));
const sections = computed(() => toolSections(props.message, toolData.value));

const isRetry = computed(() => toolData.value.name === 'retry_tool_call');
const retryHint = computed(() => retryMessage(toolData.value));

const statusIcon = computed<LucideIcon>(() => {
  switch (status.value.tone) {
    case 'success':
      return CircleCheck;
    case 'warn':
      return CircleAlert;
    case 'danger':
      return CircleAlert;
    case 'running':
      return LoaderCircle;
  }
});

const questionRequest = computed(() => parseQuestionRequest(toolData.value));

const feedbackComment = computed(() => {
  const record = isRecord(toolData.value.result) ? toolData.value.result : null;
  const comment = record?.comment;
  return typeof comment === 'string' ? comment : '';
});

function emitDecision(approved: boolean, comment: string) {
  emit('tool-decision', props.message.tool_call_id || '', approved, comment);
}

function submitAnswers(
  answers: Array<{ id: string; prompt: string; answer: unknown; skipped: boolean }>,
) {
  emit('tool-decision', props.message.tool_call_id || '', false, JSON.stringify({ answers }));
}
</script>
