<template>
  <div>
    <LoadingSpinner v-if="!stats" message="Loading stats..." color="violet" />
    <div v-else class="space-y-4">
      <div class="grid gap-3 sm:grid-cols-3">
        <StatCard :value="stats.total" label="Questions" variant="dark" color="violet" />
        <StatCard :value="stats.paper_count" label="Papers" variant="primary" color="violet" />
        <StatCard
          :value="Object.keys(stats.by_tags).length"
          label="Tags"
          variant="dark"
          color="violet"
        />
      </div>

      <div class="grid gap-3 sm:grid-cols-3">
        <div
          class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
        >
          <h3
            class="mb-2 text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
          >
            By type
          </h3>
          <div
            v-for="(count, type) in stats.by_type"
            :key="type"
            class="flex items-center justify-between py-0.5 text-xs"
          >
            <span class="font-semibold text-slate-600 dark:text-slate-300">{{
              String(type).replace('_', ' ')
            }}</span>
            <span class="font-black text-slate-900 dark:text-white">{{ count }}</span>
          </div>
          <p v-if="!Object.keys(stats.by_type).length" class="text-xs text-slate-400">
            No questions yet.
          </p>
        </div>
        <div
          class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
        >
          <h3
            class="mb-2 text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
          >
            By difficulty
          </h3>
          <div
            v-for="level in ['easy', 'medium', 'hard']"
            :key="level"
            class="flex items-center justify-between py-0.5 text-xs"
          >
            <span class="font-semibold text-slate-600 capitalize dark:text-slate-300">{{
              level
            }}</span>
            <span class="font-black text-slate-900 dark:text-white">{{
              stats.by_difficulty[level] ?? 0
            }}</span>
          </div>
        </div>
        <div
          class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
        >
          <h3
            class="mb-2 text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
          >
            By tag
          </h3>
          <div
            v-for="(count, tag) in stats.by_tags"
            :key="tag"
            class="flex items-center justify-between py-0.5 text-xs"
          >
            <span class="font-semibold text-slate-600 dark:text-slate-300">{{ tag }}</span>
            <span class="font-black text-slate-900 dark:text-white">{{ count }}</span>
          </div>
          <p v-if="!Object.keys(stats.by_tags).length" class="text-xs text-slate-400">
            No tags yet.
          </p>
        </div>
      </div>

      <div
        class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-slate-700 dark:bg-slate-800/60"
      >
        <h3
          class="mb-2 text-xs font-black tracking-wide text-slate-500 uppercase dark:text-slate-400"
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
        <div v-else class="space-y-1.5">
          <button
            v-for="paper in recentPapers"
            :key="paper.id"
            type="button"
            class="flex w-full items-center justify-between rounded-lg border border-gray-100 px-3 py-2 text-left transition hover:border-violet-300 hover:bg-violet-50/50 dark:border-slate-700 dark:hover:border-violet-700 dark:hover:bg-violet-900/10"
            @click="$emit('openPaper', paper.id)"
          >
            <span class="text-xs font-bold text-slate-700 dark:text-slate-200">{{
              paper.title
            }}</span>
            <span class="text-[11px] text-slate-400"
              >{{ paper.question_count }} questions · {{ formatDate(paper.created_at) }}</span
            >
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import LoadingSpinner from '../ui/LoadingSpinner.vue'
import StatCard from '../ui/StatCard.vue'
import EmptyState from '../ui/EmptyState.vue'
import type { QuestionPaper, QuizStats } from '../../types'

const props = defineProps<{
  stats: QuizStats | null
  papers: QuestionPaper[]
}>()

defineEmits<{ openPaper: [id: number] }>()

const recentPapers = computed(() => props.papers.slice(0, 5))

const formatDate = (iso: string) => new Date(iso).toLocaleDateString()
</script>
