<template>
  <div class="mx-auto max-w-5xl space-y-4 sm:space-y-6">
    <PresetBar @apply="applyPreset" />

    <form
      class="space-y-5 rounded-2xl border border-slate-200/80 bg-white p-4 shadow-sm sm:space-y-6 sm:p-7 dark:border-slate-800 dark:bg-slate-900"
      @submit.prevent="submit"
    >
      <!-- Title -->
      <div class="space-y-1.5">
        <label
          class="block text-xs font-bold tracking-wider text-slate-700 uppercase dark:text-slate-300"
        >
          Paper Title <span class="text-rose-500">*</span>
        </label>
        <input
          v-model="title"
          type="text"
          required
          placeholder="e.g. SSC CGL Full Length Mock Test 1"
          class="w-full rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm text-slate-900 shadow-sm transition focus:border-violet-500 focus:ring-2 focus:ring-violet-500/20 focus:outline-none dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
        />
      </div>

      <!-- Sections -->
      <div class="space-y-4">
        <div
          class="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 pb-3 dark:border-slate-800"
        >
          <div>
            <h4 class="text-sm font-bold text-slate-900 dark:text-slate-100">
              Paper Sections & Criteria
            </h4>
            <p class="text-xs text-slate-500 dark:text-slate-400">
              Customize question counts, tags, subjects, topics, and difficulties per section
            </p>
          </div>

          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-xl bg-violet-50 px-3.5 py-2 text-xs font-bold text-violet-700 transition hover:bg-violet-100 dark:bg-violet-900/40 dark:text-violet-300 dark:hover:bg-violet-900/60"
            @click="addSection"
          >
            <Plus class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
            Add Section
          </button>
        </div>

        <div class="space-y-4">
          <SectionCard
            v-for="(section, i) in sections"
            :key="i"
            ref="sectionCards"
            :section="section"
            :index="i"
            :can-remove="sections.length > 1"
            :vocabulary="vocabulary"
            @update:section="(patch) => Object.assign(sections[i], patch)"
            @duplicate="duplicateSection(i)"
            @remove="removeSection(i)"
          />
        </div>
      </div>

      <!-- Summary & submit -->
      <div
        class="flex flex-col gap-4 rounded-2xl bg-slate-100/80 p-4 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between dark:bg-slate-800/80"
      >
        <div class="flex items-center gap-3">
          <span
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-violet-600 text-sm font-black text-white tabular-nums shadow-sm"
          >
            {{ totalCount }}
          </span>
          <div>
            <p class="text-xs font-bold text-slate-900 dark:text-slate-100">
              Total Requested Questions
            </p>
            <p class="text-[11px] text-slate-500 dark:text-slate-400">
              Across {{ sections.length }} section{{ sections.length === 1 ? '' : 's' }}
            </p>
          </div>
        </div>

        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-4">
          <label
            class="flex cursor-pointer items-center gap-2 text-xs font-medium text-slate-700 dark:text-slate-300"
          >
            <input
              v-model="autoAttempt"
              type="checkbox"
              class="h-4 w-4 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-slate-600 dark:bg-slate-800"
            />
            <span>Attempt online right after generating</span>
          </label>

          <button
            type="submit"
            :disabled="isGenerating || totalCount <= 0"
            class="inline-flex items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-violet-600 to-indigo-600 px-6 py-2.5 text-xs font-bold text-white shadow-lg transition hover:from-violet-700 hover:to-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <LoaderCircle v-if="isGenerating" class="h-4 w-4 animate-spin" aria-hidden="true" />
            <WandSparkles v-else class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
            {{ isGenerating ? 'Generating Paper…' : 'Generate Paper' }}
          </button>
        </div>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { LoaderCircle, Plus, WandSparkles } from '@lucide/vue';
import type { QuestionPaperSection, TagVocabulary } from '../../../../types';
import PresetBar, { type PaperPreset } from './PresetBar.vue';
import SectionCard from './SectionCard.vue';

defineProps<{
  vocabulary?: TagVocabulary | null;
  isGenerating?: boolean;
}>();

const emit = defineEmits<{
  generate: [input: { title: string; sections: QuestionPaperSection[]; autoAttempt?: boolean }];
}>();

const title = ref('');
const autoAttempt = ref(true);
const sectionCards = ref<Array<{ flush: () => void } | null>>([]);

const blankSection = (): QuestionPaperSection =>
  ({
    tags: [],
    subject: '',
    topic: '',
    type: undefined,
    difficulty: undefined,
    count: 10,
  }) as QuestionPaperSection;

const sections = reactive<QuestionPaperSection[]>([blankSection()]);

const addSection = () => sections.push(blankSection());

const removeSection = (index: number) => {
  if (sections.length <= 1) return;
  sections.splice(index, 1);
};

const duplicateSection = (index: number) => {
  const target = sections[index];
  sections.push({
    tags: target.tags ? [...target.tags] : [],
    subject: target.subject || '',
    topic: target.topic || '',
    type: target.type,
    difficulty: target.difficulty,
    count: target.count || 10,
  } as QuestionPaperSection);
};

function applyPreset(preset: PaperPreset) {
  title.value = preset.title;
  sections.splice(
    0,
    sections.length,
    ...preset.sections.map(
      (s) =>
        ({
          tags: [],
          subject: '',
          topic: '',
          type: undefined,
          difficulty: s.difficulty,
          count: s.count,
        }) as QuestionPaperSection,
    ),
  );
}

const totalCount = computed(() => sections.reduce((sum, s) => sum + (s.count || 0), 0));

const submit = () => {
  // Pick up half-typed tags in every section before building the payload.
  sectionCards.value.forEach((card) => card?.flush());

  const cleaned = sections.map((s) => {
    const out: QuestionPaperSection = { count: s.count };
    if (s.tags?.length) out.tags = s.tags;
    if (s.subject) out.subject = s.subject;
    if (s.topic) out.topic = s.topic;
    if (s.type) out.type = s.type;
    if (s.difficulty) out.difficulty = s.difficulty;
    return out;
  });

  emit('generate', {
    title: title.value.trim(),
    sections: cleaned,
    autoAttempt: autoAttempt.value,
  });
  title.value = '';
};
</script>
