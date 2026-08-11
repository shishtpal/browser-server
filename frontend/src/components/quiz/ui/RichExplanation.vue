<template>
  <div class="relative">
    <button
      type="button"
      class="absolute top-1 right-1 grid h-6 w-6 place-items-center rounded-md text-slate-400 transition hover:bg-violet-50 hover:text-violet-600 dark:hover:bg-violet-900/30 dark:hover:text-violet-400"
      :class="{ 'text-violet-600 dark:text-violet-400': mathEnabled }"
      :aria-pressed="mathEnabled"
      :title="mathEnabled ? 'Disable math rendering (MathJax)' : 'Enable math rendering (MathJax)'"
      @click="mathEnabled = !mathEnabled"
    >
      <Sigma class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
    </button>
    <div
      ref="contentEl"
      v-html="rendered"
      :class="size === 'sm' ? 'text-xs' : 'text-sm'"
      class="overflow-x-auto pr-8 leading-relaxed text-slate-600 dark:text-slate-300"
      @click="copyCodeBlock"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { Sigma } from '@lucide/vue';
import { renderMarkdown, typesetMath } from '@browser-server/shared-markdown';
import { copyToClipboard } from '../../../utils/copyToClipboard';

const props = defineProps<{
  markdown: string;
  size?: 'sm' | 'md';
}>();

/** Math rendering is opt-in per instance (MathJax loads from CDN on demand). */
const mathEnabled = ref(false);

const rendered = computed(() => renderMarkdown(props.markdown, { math: mathEnabled.value }));

const contentEl = ref<HTMLElement | null>(null);

async function runTypeset() {
  await nextTick();
  if (contentEl.value && mathEnabled.value) await typesetMath(contentEl.value);
}

watch([rendered, mathEnabled], () => void runTypeset(), { immediate: true });

/** Copy controls are emitted by the shared Markdown renderer for fenced code blocks. */
function copyCodeBlock(event: MouseEvent) {
  if (!(event.target instanceof Element)) return;
  const button = event.target.closest<HTMLButtonElement>('[data-copy-code]');
  const code = button?.parentElement?.querySelector<HTMLElement>('code');
  if (code) void copyToClipboard(code.innerText);
}
</script>
