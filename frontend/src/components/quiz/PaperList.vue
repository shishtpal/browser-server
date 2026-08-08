<template>
  <div>
    <EmptyState
      v-if="papers.length === 0"
      title="No papers yet"
      description="Generate a paper from the Generate Paper tab."
      icon="search"
      color="violet"
    />
    <div v-else class="space-y-2">
      <div
        v-for="paper in papers"
        :key="paper.id"
        class="flex items-center gap-3 rounded-xl border border-gray-200 bg-white p-3 shadow-sm transition-colors dark:border-slate-700 dark:bg-slate-800/60"
      >
        <button type="button" class="flex-1 text-left" @click="$emit('open', paper.id)">
          <p class="text-sm font-bold text-slate-800 dark:text-slate-100">{{ paper.title }}</p>
          <p class="text-[11px] text-slate-400">
            {{ paper.question_count }} questions · {{ paper.sections.length }} section(s) ·
            {{ formatDate(paper.created_at) }}
          </p>
        </button>
        <button
          type="button"
          class="rounded-md px-2 py-1 text-[11px] font-semibold text-rose-600 transition hover:bg-rose-50 dark:text-rose-400 dark:hover:bg-rose-900/30"
          @click="$emit('delete', paper.id)"
        >
          Delete
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { QuestionPaper } from '../../types';
import EmptyState from '../ui/EmptyState.vue';

defineProps<{ papers: QuestionPaper[] }>();
defineEmits<{ open: [id: number]; delete: [id: number] }>();

const formatDate = (iso: string) => new Date(iso).toLocaleString();
</script>
