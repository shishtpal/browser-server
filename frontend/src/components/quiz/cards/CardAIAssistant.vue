<template>
  <section
    v-if="ai.available.value"
    class="rounded-xl border border-violet-200/70 bg-violet-50/50 dark:border-violet-800/60 dark:bg-violet-950/20"
    aria-label="Ask AI about this question"
  >
    <!-- Collapsed trigger / expanded header -->
    <button
      type="button"
      class="flex w-full items-center gap-2 px-3.5 py-2.5 text-left text-sm font-semibold text-violet-700 transition hover:text-violet-800 dark:text-violet-300 dark:hover:text-violet-200"
      :aria-expanded="isOpen"
      @click="toggle"
    >
      <Sparkles class="h-4 w-4 shrink-0" :stroke-width="2.25" aria-hidden="true" />
      <span class="flex-1">Ask AI to explain or cross-check this question</span>
      <span
        v-if="ai.activeModelLabel.value"
        class="hidden max-w-40 truncate rounded-md bg-violet-100 px-1.5 py-0.5 text-[10px] font-bold text-violet-600 sm:inline dark:bg-violet-900/50 dark:text-violet-300"
        :title="`Active model: ${ai.activeModelLabel.value}`"
      >
        {{ ai.activeModelLabel.value }}
      </span>
      <ChevronDown
        class="h-4 w-4 shrink-0 transition-transform duration-200"
        :class="{ 'rotate-180': isOpen }"
        :stroke-width="2.25"
        aria-hidden="true"
      />
    </button>

    <div v-if="isOpen" class="border-t border-violet-200/60 px-3.5 pt-3 pb-3.5 dark:border-violet-800/50">
      <!-- Actions + settings -->
      <div class="relative flex flex-wrap items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          :disabled="ai.isStreaming.value"
          @click="ai.ask(question, 'explain')"
        >
          <span class="inline-flex items-center gap-1.5">
            <Sparkles class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
            Explain
          </span>
        </Button>

        <Button
          variant="secondary"
          size="sm"
          :disabled="ai.isStreaming.value"
          @click="ai.ask(question, 'crosscheck')"
        >
          <span class="inline-flex items-center gap-1.5">
            <ShieldCheck class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
            Cross-check answer
          </span>
        </Button>

        <Button v-if="ai.isStreaming.value" variant="danger" size="sm" @click="ai.stop()">
          <span class="inline-flex items-center gap-1.5">
            <Square class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
            Stop
          </span>
        </Button>

        <span
          v-if="ai.isStreaming.value && !ai.content.value"
          class="text-xs text-violet-500 italic dark:text-violet-300"
        >
          Thinking…
        </span>

        <!-- Settings toggle -->
        <button
          type="button"
          class="ml-auto grid h-7 w-7 place-items-center rounded-lg text-violet-400 transition hover:bg-violet-100 hover:text-violet-600 dark:text-violet-400 dark:hover:bg-violet-900/40 dark:hover:text-violet-300"
          :class="{ 'bg-violet-100 text-violet-600 dark:bg-violet-900/40 dark:text-violet-300': showSettings }"
          title="Choose provider and model"
          aria-label="Choose provider and model"
          :aria-expanded="showSettings"
          @click="showSettings = !showSettings"
        >
          <Settings class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>

        <!-- Settings popover -->
        <transition
          enter-active-class="transition ease-out duration-150"
          enter-from-class="opacity-0 translate-y-1 scale-95"
          enter-to-class="opacity-100 translate-y-0 scale-100"
          leave-active-class="transition ease-in duration-100"
          leave-from-class="opacity-100 translate-y-0 scale-100"
          leave-to-class="opacity-0 translate-y-1 scale-95"
        >
          <div
            v-if="showSettings"
            class="absolute bottom-full right-0 z-50 mb-1 w-64 space-y-2.5 rounded-lg border border-slate-200 bg-white p-3 shadow-lg dark:border-white/10 dark:bg-slate-900"
          >
            <label class="block space-y-1">
              <span class="text-[10px] font-bold uppercase tracking-wide text-slate-500 dark:text-slate-400">Provider</span>
              <SearchableSelect v-model="ai.provider.value" :items="ai.providerItems.value" class="w-full" />
            </label>
            <label class="block space-y-1">
              <span class="text-[10px] font-bold uppercase tracking-wide text-slate-500 dark:text-slate-400">Model</span>
              <SearchableSelect
                v-model="ai.model.value"
                :items="ai.modelItems.value"
                :searchable="true"
                search-placeholder="Search models..."
                placeholder="Select a model"
                class="w-full"
              />
            </label>
            <p class="text-[10px] leading-snug text-slate-400 dark:text-slate-500">
              Saved as the default for Ask AI on this device.
            </p>
          </div>
        </transition>
        <div v-if="showSettings" class="fixed inset-0 z-40" @click="showSettings = false" />
      </div>

      <!-- Error -->
      <p
        v-if="ai.error.value"
        class="mt-2.5 rounded-lg border border-rose-200 bg-rose-50 px-2.5 py-2 text-xs font-medium text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/30 dark:text-rose-300"
      >
        {{ ai.error.value }}
      </p>

      <!-- Output -->
      <div
        v-if="ai.content.value"
        class="mt-3 rounded-lg border border-violet-200/60 bg-white/80 p-3 dark:border-violet-800/50 dark:bg-slate-900/60"
      >
        <p class="mb-1.5 text-[10px] font-bold uppercase tracking-wide text-violet-500 dark:text-violet-300">
          {{ ai.mode.value === 'crosscheck' ? 'Cross-check result' : 'AI explanation' }}
        </p>
        <!-- AI-generated markdown; renderMarkdown HTML-escapes input -->
        <div class="text-sm leading-relaxed text-slate-700 dark:text-slate-200" v-html="rendered" />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { ChevronDown, Settings, ShieldCheck, Sparkles, Square } from '@lucide/vue';
import { renderMarkdown } from '@browser-server/shared-markdown';
import type { QuestionResponse } from '../../../types';
import Button from '../../ui/Button.vue';
import SearchableSelect from '../../ui/SearchableSelect.vue';
import { useQuestionAI } from '../composables/useQuestionAI';

const props = defineProps<{
  question: QuestionResponse;
}>();

const ai = useQuestionAI();

const isOpen = ref(false);
const showSettings = ref(false);

// Re-render markdown on every streamed delta.
const rendered = computed(() => renderMarkdown(ai.content.value, { math: false }));

function toggle() {
  isOpen.value = !isOpen.value;
  if (!isOpen.value) {
    showSettings.value = false;
    ai.clear(); // abort + drop the ephemeral conversation
  }
}

// New card → reset any previous run.
watch(
  () => props.question.id,
  () => ai.clear(),
);

onMounted(() => ai.init());
</script>
