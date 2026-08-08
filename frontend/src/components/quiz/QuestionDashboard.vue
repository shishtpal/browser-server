<template>
  <div>
    <LoadingSpinner v-if="!stats" message="Loading stats..." color="violet" />
    <div v-else class="space-y-4">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <StatCard :value="stats.total" label="Questions" variant="dark" color="violet" />
        <StatCard :value="stats.paper_count" label="Papers" variant="primary" color="violet" />
        <StatCard
          :value="Object.keys(stats.by_tags).length"
          label="Tags"
          variant="dark"
          color="violet"
        />
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <div
          class="flex flex-col justify-between rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
        >
          <h3
            class="mb-2 text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
          >
            By type
          </h3>
          <div class="space-y-1">
            <div
              v-for="(count, type) in stats.by_type"
              :key="type"
              class="flex items-center justify-between py-1 text-xs"
            >
              <span class="font-semibold text-slate-600 dark:text-slate-300">{{
                String(type).replace('_', ' ')
              }}</span>
              <span
                class="rounded bg-slate-100 px-2 py-0.5 font-black text-slate-900 dark:bg-slate-700 dark:text-white"
                >{{ count }}</span
              >
            </div>
            <p v-if="!Object.keys(stats.by_type).length" class="text-xs text-slate-400">
              No questions yet.
            </p>
          </div>
        </div>

        <div
          class="flex flex-col justify-between rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
        >
          <h3
            class="mb-2 text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
          >
            By difficulty
          </h3>
          <div class="space-y-1">
            <div
              v-for="level in ['easy', 'medium', 'hard']"
              :key="level"
              class="flex items-center justify-between py-1 text-xs"
            >
              <span class="font-semibold text-slate-600 capitalize dark:text-slate-300">{{
                level
              }}</span>
              <span
                class="rounded bg-slate-100 px-2 py-0.5 font-black text-slate-900 dark:bg-slate-700 dark:text-white"
                >{{ stats.by_difficulty[level] ?? 0 }}</span
              >
            </div>
          </div>
        </div>

        <div
          class="flex flex-col rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
        >
          <div class="mb-2 flex items-center justify-between">
            <h3
              class="text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
            >
              By tag
            </h3>
            <span v-if="sortedTags.length" class="text-[10px] font-semibold text-slate-400">
              Sorted by count
            </span>
          </div>

          <div
            v-if="sortedTags.length"
            class="max-h-40 scrollbar-thin scrollbar-thumb-slate-300 space-y-1 overflow-y-auto pr-1 dark:scrollbar-thumb-slate-600"
          >
            <div
              v-for="item in sortedTags"
              :key="item.tag"
              class="flex items-center justify-between py-1 text-xs"
            >
              <span
                class="truncate font-semibold text-slate-600 dark:text-slate-300"
                :title="item.tag"
              >
                {{ item.tag }}
              </span>
              <span
                class="ml-2 shrink-0 rounded bg-violet-50 px-2 py-0.5 font-black text-violet-700 dark:bg-violet-900/40 dark:text-violet-300"
              >
                {{ item.count }}
              </span>
            </div>
          </div>
          <p v-else class="text-xs text-slate-400">No tags yet.</p>
        </div>
      </div>

      <div
        class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
      >
        <h3
          class="mb-3 text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
        >
          Recent papers
        </h3>
        <EmptyState
          v-if="recentPapers.length === 0"
          title="No papers yet"
          description="Generate a paper from the Generate Paper tab."
          icon="search"
          color="violet"
        />
        <div v-else class="space-y-2">
          <button
            v-for="paper in recentPapers"
            :key="paper.id"
            type="button"
            class="flex w-full flex-col justify-between gap-1 rounded-lg border border-gray-100 p-3 text-left transition hover:border-violet-300 hover:bg-violet-50/50 sm:flex-row sm:items-center sm:gap-2 dark:border-slate-700 dark:hover:border-violet-700 dark:hover:bg-violet-900/10"
            @click="$emit('openPaper', paper.id)"
          >
            <span class="truncate text-xs font-bold text-slate-700 dark:text-slate-200">
              {{ paper.title }}
            </span>
            <span class="shrink-0 text-[11px] font-medium text-slate-400">
              {{ paper.question_count }} questions · {{ formatDate(paper.created_at) }}
            </span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { QuestionPaper, QuizStats } from '../../types';
import { computed } from 'vue';
import StatCard from '../ui/StatCard.vue';
import EmptyState from '../ui/EmptyState.vue';
import LoadingSpinner from '../ui/LoadingSpinner.vue';

const props = defineProps<{
  stats: QuizStats | null;
  papers: QuestionPaper[];
}>();

defineEmits<{ openPaper: [id: number] }>();

const recentPapers = computed(() => props.papers.slice(0, 5));

const sortedTags = computed(() => {
  if (!props.stats?.by_tags) return [];
  return Object.entries(props.stats.by_tags)
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag));
});

const formatDate = (iso: string) => new Date(iso).toLocaleDateString();
</script>
