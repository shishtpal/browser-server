<template>
  <div class="flex min-h-0 flex-1 flex-col">
    <!-- Editor sub-header -->
    <div
      class="flex items-center justify-between border-b border-slate-200 px-4 py-2.5 dark:border-white/10"
    >
      <button
        class="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-[0.8rem] font-medium text-slate-600 transition hover:bg-slate-100 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-white/10 dark:hover:text-white"
        type="button"
        @click="$emit('back')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19 12H5m0 0 6 6m-6-6 6-6" />
        </svg>
        Back to prompts
      </button>
      <span class="text-[0.72rem] text-slate-400 dark:text-slate-500">
        {{ draft.id ? `Editing · last saved ${formatShortDate(draft.updated_at)}` : 'New prompt' }}
      </span>
    </div>

    <!-- Form -->
    <form
      class="chat-scroll flex min-h-0 flex-1 flex-col overflow-auto px-4 py-4"
      @submit.prevent="$emit('save')"
    >
      <div class="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-4">
        <div>
          <label
            class="mb-1 block text-[0.68rem] font-semibold tracking-wider text-slate-500 uppercase dark:text-slate-400"
            >Title</label
          >
          <input
            ref="titleInputRef"
            :value="draft.title"
            class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-[0.85rem] text-slate-900 transition outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/15 dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
            placeholder="Prompt title"
            required
            @input="$emit('update:title', ($event.target as HTMLInputElement).value)"
          />
        </div>

        <div>
          <label
            class="mb-1 block text-[0.68rem] font-semibold tracking-wider text-slate-500 uppercase dark:text-slate-400"
            >Description</label
          >
          <input
            :value="draft.description"
            class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-[0.85rem] text-slate-900 transition outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/15 dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
            placeholder="Short description"
            @input="$emit('update:description', ($event.target as HTMLInputElement).value)"
          />
        </div>

        <div>
          <label
            class="mb-1 block text-[0.68rem] font-semibold tracking-wider text-slate-500 uppercase dark:text-slate-400"
            >Tags</label
          >
          <input
            :value="tagsInput"
            class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-[0.85rem] text-slate-900 transition outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/15 dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
            placeholder="system, assistant, coding"
            @input="$emit('update:tagsInput', ($event.target as HTMLInputElement).value)"
          />
          <div v-if="parsedTags.length" class="mt-2 flex flex-wrap gap-1">
            <span
              v-for="tag in parsedTags"
              :key="tag"
              class="rounded-full bg-indigo-50 px-2 py-0.5 text-[0.68rem] font-medium text-indigo-600 dark:bg-indigo-500/15 dark:text-indigo-300"
              >{{ tag }}</span
            >
          </div>
        </div>

        <div class="flex min-h-[300px] flex-1 flex-col">
          <div class="mb-1 flex items-center justify-between">
            <label
              class="block text-[0.68rem] font-semibold tracking-wider text-slate-500 uppercase dark:text-slate-400"
              >Prompt content</label
            >
            <span class="text-[0.68rem] text-slate-400 tabular-nums dark:text-slate-500"
              >{{ draft.content.length }} chars</span
            >
          </div>
          <textarea
            :value="draft.content"
            class="chat-scroll h-full min-h-[300px] w-full flex-1 resize-none rounded-lg border border-slate-200 bg-white px-3 py-3 font-mono text-[0.82rem] leading-6 text-slate-900 transition outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/15 dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
            placeholder="Write or edit your prompt here…"
            required
            @input="$emit('update:content', ($event.target as HTMLTextAreaElement).value)"
          />
        </div>
      </div>
    </form>

    <!-- Footer -->
    <div
      class="flex items-center justify-between border-t border-slate-200 bg-white/60 px-4 py-3 dark:border-white/10 dark:bg-slate-900/50"
    >
      <button
        v-if="draft.id"
        class="inline-flex items-center gap-1.5 rounded-lg px-3 py-2 text-[0.8rem] font-semibold text-red-500 transition hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-500/10"
        type="button"
        @click="$emit('delete')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M19 7l-.867 12.142A2 2 0 0 1 16.138 21H7.862a2 2 0 0 1-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v3M4 7h16"
          />
        </svg>
        Delete
      </button>
      <span v-else></span>

      <div class="flex items-center gap-2">
        <button
          class="rounded-lg border border-slate-200 px-3 py-2 text-[0.8rem] font-semibold text-slate-600 transition hover:bg-slate-50 dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/10"
          type="button"
          @click="$emit('back')"
        >
          Cancel
        </button>
        <button
          class="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-4 py-2 text-[0.8rem] font-semibold text-white shadow-sm transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          type="button"
          :disabled="!canSave || isSaving"
          @click="$emit('save')"
        >
          <svg v-if="isSaving" class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            />
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 0 1 8-8v4a4 4 0 0 0-4 4H4z"
            />
          </svg>
          {{ isSaving ? 'Saving…' : 'Save prompt' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PromptDraft } from '../../composables/usePromptManager';
import { nextTick, onMounted, ref } from 'vue';
import { formatShortDate } from './format';

defineProps<{
  draft: PromptDraft;
  tagsInput: string;
  parsedTags: string[];
  canSave: boolean;
  isSaving: boolean;
}>();

defineEmits<{
  'update:title': [value: string];
  'update:description': [value: string];
  'update:tagsInput': [value: string];
  'update:content': [value: string];
  save: [];
  back: [];
  delete: [];
}>();

const titleInputRef = ref<HTMLInputElement | null>(null);

onMounted(() => {
  nextTick(() => titleInputRef.value?.focus());
});
</script>
