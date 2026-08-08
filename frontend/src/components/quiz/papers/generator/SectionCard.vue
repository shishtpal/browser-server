<template>
  <div
    class="relative rounded-2xl border border-slate-200 bg-slate-50/60 p-4 transition sm:p-5 dark:border-slate-800 dark:bg-slate-900/50"
  >
    <!-- Section header -->
    <div
      class="mb-4 flex items-center justify-between gap-2 border-b border-slate-200/60 pb-3 dark:border-slate-800"
    >
      <div class="flex min-w-0 items-center gap-2">
        <span
          class="flex h-6 w-6 shrink-0 items-center justify-center rounded-lg bg-violet-600 text-xs font-black text-white"
        >
          {{ index + 1 }}
        </span>
        <span class="truncate text-xs font-bold text-slate-800 dark:text-slate-200">
          Section {{ index + 1 }}
        </span>
      </div>

      <div class="flex shrink-0 items-center gap-1">
        <button
          type="button"
          class="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[11px] font-semibold text-slate-600 transition hover:bg-slate-200 dark:text-slate-300 dark:hover:bg-slate-800"
          @click="$emit('duplicate')"
        >
          <Copy class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
          <span class="hidden sm:inline">Duplicate</span>
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1 rounded-lg px-2.5 py-1.5 text-[11px] font-bold text-rose-600 transition hover:bg-rose-50 disabled:opacity-30 dark:text-rose-400 dark:hover:bg-rose-950/40"
          :disabled="!canRemove"
          @click="$emit('remove')"
        >
          <Trash2 class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
          <span class="hidden sm:inline">Remove</span>
        </button>
      </div>
    </div>

    <!-- Criteria grid -->
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <!-- Tags -->
      <div class="space-y-1 sm:col-span-2 lg:col-span-3">
        <label class="block text-[11px] font-bold text-slate-600 dark:text-slate-400">
          Tags (filter questions by tags)
        </label>
        <TagInput
          ref="tagInput"
          :model-value="section.tags ?? []"
          :suggestions="tagSuggestions"
          :list-id="tagListId"
          placeholder="Type tag (e.g. SSC, RRB)…"
          @update:model-value="update({ tags: $event })"
        />
      </div>

      <div class="space-y-1">
        <label class="block text-[11px] font-bold text-slate-600 dark:text-slate-400"
          >Subject</label
        >
        <select :value="section.subject" :class="selectClass" @change="onSelect('subject', $event)">
          <option value="">Any Subject</option>
          <option v-for="v in vocabulary?.subjects ?? []" :key="v" :value="v">{{ v }}</option>
        </select>
      </div>

      <div class="space-y-1">
        <label class="block text-[11px] font-bold text-slate-600 dark:text-slate-400">Topic</label>
        <select :value="section.topic" :class="selectClass" @change="onSelect('topic', $event)">
          <option value="">Any Topic</option>
          <option v-for="v in vocabulary?.topics ?? []" :key="v" :value="v">{{ v }}</option>
        </select>
      </div>

      <div class="space-y-1">
        <label class="block text-[11px] font-bold text-slate-600 dark:text-slate-400">
          Question Format
        </label>
        <select :value="section.type" :class="selectClass" @change="onSelect('type', $event)">
          <option value="">Any Format</option>
          <option value="single_choice">Single Choice</option>
          <option value="multiple_choice">Multiple Choice</option>
          <option value="input">Free Text Input</option>
          <option value="chronology">Chronology</option>
        </select>
      </div>

      <div class="space-y-1">
        <label class="block text-[11px] font-bold text-slate-600 dark:text-slate-400">
          Difficulty Level
        </label>
        <select
          :value="section.difficulty"
          :class="selectClass"
          @change="onSelect('difficulty', $event)"
        >
          <option value="">Any Difficulty</option>
          <option value="easy">Easy</option>
          <option value="medium">Medium</option>
          <option value="hard">Hard</option>
        </select>
      </div>

      <div class="space-y-1">
        <label class="block text-[11px] font-bold text-slate-600 dark:text-slate-400">
          Number of Questions <span class="text-rose-500">*</span>
        </label>
        <input
          :value="section.count"
          type="number"
          min="1"
          max="200"
          required
          placeholder="10"
          :class="[selectClass, 'font-bold']"
          @input="update({ count: Number(($event.target as HTMLInputElement).value) || 0 })"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { Copy, Trash2 } from '@lucide/vue';
import type { QuestionPaperSection, TagVocabulary } from '../../../../types';
import TagInput from '../../ui/TagInput.vue';

const props = defineProps<{
  section: QuestionPaperSection;
  index: number;
  canRemove: boolean;
  vocabulary?: TagVocabulary | null;
}>();

const emit = defineEmits<{
  'update:section': [patch: Partial<QuestionPaperSection>];
  duplicate: [];
  remove: [];
}>();

const tagInput = ref<{ flush: () => void } | null>(null);

const tagSuggestions = computed(() => props.vocabulary?.tags ?? []);
const tagListId = computed(() => `paper-section-tags-${props.index}`);

const selectClass =
  'w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-xs text-slate-900 shadow-sm focus:border-violet-500 focus:ring-2 focus:ring-violet-500/20 focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100';

const update = (patch: Partial<QuestionPaperSection>) => emit('update:section', patch);

const onSelect = (key: 'subject' | 'topic' | 'type' | 'difficulty', e: Event) => {
  const value = (e.target as HTMLSelectElement).value || undefined;
  update({ [key]: value } as Partial<QuestionPaperSection>);
};

/** Flush any half-typed tag into the model (called by the parent on submit). */
const flush = () => tagInput.value?.flush();

defineExpose({ flush });
</script>
