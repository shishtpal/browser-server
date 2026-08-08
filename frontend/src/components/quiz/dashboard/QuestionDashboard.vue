<template>
  <div>
    <LoadingSpinner v-if="!stats" message="Loading stats..." color="violet" />
    <div v-else class="space-y-4">
      <!-- Headline stats -->
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <button
          v-for="card in statCards"
          :key="card.label"
          type="button"
          class="group relative flex items-center gap-3 overflow-hidden rounded-xl border border-gray-200 bg-white p-4 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-violet-300 hover:shadow-md dark:border-slate-700 dark:bg-slate-800/60 dark:hover:border-violet-700"
          @click="$emit('navigate', card.tab)"
        >
          <span
            class="grid h-11 w-11 shrink-0 place-items-center rounded-xl text-white shadow-sm"
            :class="card.bgClass"
          >
            <component :is="card.icon" class="h-5 w-5" :stroke-width="2.25" aria-hidden="true" />
          </span>
          <span class="min-w-0">
            <span class="block text-xl font-black text-slate-900 tabular-nums dark:text-white">
              {{ card.value }}
            </span>
            <span
              class="block truncate text-[11px] font-bold tracking-wide text-slate-500 uppercase dark:text-slate-400"
            >
              {{ card.label }}
            </span>
          </span>
          <ArrowRight
            class="ml-auto h-4 w-4 shrink-0 text-slate-300 transition group-hover:translate-x-0.5 group-hover:text-violet-500 dark:text-slate-600"
            :stroke-width="2.5"
            aria-hidden="true"
          />
        </button>
      </div>

      <!-- Breakdowns -->
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
        <DashboardBreakdownCard
          title="By type"
          :icon="ListChecks"
          :items="typeItems"
          empty-text="No questions yet."
        />
        <DashboardBreakdownCard
          title="By difficulty"
          :icon="Gauge"
          :items="difficultyItems"
          empty-text="No questions yet."
        />
        <DashboardBreakdownCard
          title="By tag"
          :icon="Tags"
          :items="tagItems"
          hint="Sorted by count"
          empty-text="No tags yet."
          class="md:col-span-2 xl:col-span-1"
        />
      </div>

      <RecentPapersList :papers="papers" @open="$emit('openPaper', $event)" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  ArrowRight,
  FileText,
  Gauge,
  Layers,
  ListChecks,
  Tags,
  type LucideIcon,
} from '@lucide/vue';
import type { QuestionPaper, QuizStats } from '../../../types';
import LoadingSpinner from '../../ui/LoadingSpinner.vue';
import DashboardBreakdownCard, { type BreakdownItem } from './DashboardBreakdownCard.vue';
import RecentPapersList from './RecentPapersList.vue';
import type { QuizTab } from '../composables/useQuizPage';
import { DIFFICULTY_META, QUESTION_TYPE_META } from '../quizFormat';

const props = defineProps<{
  stats: QuizStats | null;
  papers: QuestionPaper[];
}>();

defineEmits<{
  openPaper: [id: number];
  navigate: [tab: QuizTab];
}>();

const statCards = computed<
  Array<{
    label: string;
    value: number;
    icon: LucideIcon;
    bgClass: string;
    tab: QuizTab;
  }>
>(() => [
  {
    label: 'Questions',
    value: props.stats?.total ?? 0,
    icon: ListChecks,
    bgClass: 'bg-gradient-to-br from-violet-500 to-indigo-600',
    tab: 'questions',
  },
  {
    label: 'Papers',
    value: props.stats?.paper_count ?? 0,
    icon: FileText,
    bgClass: 'bg-gradient-to-br from-fuchsia-500 to-purple-600',
    tab: 'papers',
  },
  {
    label: 'Tags',
    value: Object.keys(props.stats?.by_tags ?? {}).length,
    icon: Layers,
    bgClass: 'bg-gradient-to-br from-cyan-500 to-blue-600',
    tab: 'cards',
  },
]);

const typeItems = computed<BreakdownItem[]>(() =>
  Object.entries(props.stats?.by_type ?? {}).map(([type, count]) => {
    const meta = QUESTION_TYPE_META[type as keyof typeof QUESTION_TYPE_META];
    return { key: type, label: meta?.label ?? type.replaceAll('_', ' '), count, icon: meta?.icon };
  }),
);

const difficultyItems = computed<BreakdownItem[]>(() =>
  Object.values(DIFFICULTY_META).map((meta) => ({
    key: meta.label,
    label: meta.label,
    count: props.stats?.by_difficulty[meta.label.toLowerCase()] ?? 0,
    dotClass: meta.dotClass,
  })),
);

const tagItems = computed<BreakdownItem[]>(() =>
  Object.entries(props.stats?.by_tags ?? {})
    .map(([tag, count]) => ({ key: tag, label: tag, count, accent: true }))
    .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label)),
);
</script>
