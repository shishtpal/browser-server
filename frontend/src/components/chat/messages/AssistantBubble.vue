<template>
  <article
    class="group relative max-w-[96%] rounded-xl rounded-bl-sm border border-slate-200/80 bg-white px-4 py-3 shadow-sm sm:max-w-[92%] dark:border-white/10 dark:bg-slate-900/70"
  >
    <ChatThinkingBlock
      v-if="showThinking"
      :reasoning="message.reasoning"
      :streaming="message.status === 'pending' && !message.content"
    />

    <!-- Streaming placeholder -->
    <div
      v-if="
        message.status === 'pending' && !message.content && (!showThinking || !message.reasoning)
      "
      class="flex items-center gap-2 text-[0.82em] font-medium text-slate-400"
      role="status"
    >
      <LoaderCircle class="h-3.5 w-3.5 animate-spin text-indigo-500" aria-hidden="true" />
      Thinking…
    </div>

    <div
      v-else
      ref="contentEl"
      class="prose prose-slate dark:prose-invert prose-p:text-[0.92em] prose-p:leading-[1.65] prose-li:text-[0.92em] prose-headings:font-semibold prose-headings:tracking-tight prose-h1:text-[1.2em] prose-h2:text-[1.1em] prose-h3:text-[1em] prose-pre:my-2 prose-pre:rounded-lg max-w-none break-words"
      v-html="renderedContent"
      @click="copyCodeBlock"
    ></div>

    <div
      v-if="message.status === 'error'"
      class="mt-2 flex items-center gap-1.5 text-[0.82em] font-medium text-red-500"
      role="alert"
    >
      <CircleAlert class="h-3.5 w-3.5" aria-hidden="true" />
      Generation failed
    </div>
    <div
      v-if="message.status === 'cancelled'"
      class="mt-2 flex items-center gap-1.5 text-[0.82em] font-medium text-amber-500"
    >
      <StopCircle class="h-3.5 w-3.5" aria-hidden="true" />
      Stopped
    </div>

    <BubbleActions
      :include="
        ttsAvailable
          ? ['copy', 'speak', 'branch', 'math', 'delete']
          : ['copy', 'branch', 'math', 'delete']
      "
      :active="activeActions"
      :busy="busyActions"
      @action="onAction"
    />
  </article>
</template>

<script setup lang="ts">
import type { AIMessage } from '@browser-server/shared-types';
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';
import { CircleAlert, LoaderCircle, StopCircle } from '@lucide/vue';
import BubbleActions, { type BubbleActionName } from './BubbleActions.vue';
import ChatThinkingBlock from '../ChatThinkingBlock.vue';
import { renderMarkdown, typesetMath } from '@browser-server/shared-markdown';

const props = withDefaults(
  defineProps<{
    message: AIMessage;
    showThinking?: boolean;
    ttsAvailable?: boolean;
    speakBusy?: boolean;
    speakActive?: boolean;
  }>(),
  { showThinking: true, ttsAvailable: true, speakBusy: false, speakActive: false },
);

const emit = defineEmits<{
  copy: [content: string];
  delete: [messageId: string];
  branch: [messageId: string];
  speak: [payload: { messageId: string; content: string }];
}>();

const activeActions = computed(() => {
  const names: BubbleActionName[] = [];
  if (mathEnabled.value) names.push('math');
  if (props.speakActive) names.push('speak');
  return names;
});

const busyActions = computed<BubbleActionName[]>(() => (props.speakBusy ? ['speak'] : []));

/** Math rendering is opt-in per message (MathJax loads from CDN on demand). */
const mathEnabled = ref(false);

const renderedContent = computed(() =>
  renderMarkdown(props.message.content, { math: mathEnabled.value }),
);

/** Reference to the content container for MathJax typesetting. */
const contentEl = ref<HTMLElement | null>(null);

/** Re-typeset math whenever the rendered HTML changes.
 *  While the message is still streaming, updates are debounced (trailing
 *  edge) so each chunk doesn't queue behind the MathJax CDN load. */
let typesetTimer: ReturnType<typeof setTimeout> | undefined;
let contentVersion = 0;

async function runTypeset() {
  await nextTick();
  if (contentEl.value) await typesetMath(contentEl.value);
}

watch(
  [renderedContent, mathEnabled],
  () => {
    if (!mathEnabled.value) return;
    contentVersion++;
    if (props.message.status === 'pending') {
      clearTimeout(typesetTimer);
      const v = contentVersion;
      typesetTimer = setTimeout(() => {
        if (v === contentVersion) void runTypeset();
      }, 350);
    } else {
      void runTypeset();
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => clearTimeout(typesetTimer));

/** "Copy" buttons rendered by the markdown code blocks bubble up here. */
function copyCodeBlock(event: MouseEvent) {
  if (!(event.target instanceof Element)) return;
  const button = event.target.closest<HTMLButtonElement>('[data-copy-code]');
  if (!button) return;
  const code = button.parentElement?.querySelector<HTMLElement>('code');
  if (code) emit('copy', code.innerText);
}

function onAction(name: BubbleActionName) {
  if (name === 'math') mathEnabled.value = !mathEnabled.value;
  else if (name === 'copy') emit('copy', props.message.content);
  else if (name === 'branch') emit('branch', props.message.id);
  else if (name === 'speak')
    emit('speak', { messageId: props.message.id, content: props.message.content });
  else emit('delete', props.message.id);
}
</script>
