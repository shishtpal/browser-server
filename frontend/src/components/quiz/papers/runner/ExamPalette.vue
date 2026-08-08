<template>
  <!-- Mobile backdrop -->
  <Transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-30 bg-slate-950/50 backdrop-blur-xs md:hidden"
      @click="$emit('close')"
    ></div>
  </Transition>

  <aside
    class="w-72 max-w-[85vw] shrink-0 border-l border-slate-200 bg-white p-4 md:block dark:border-slate-800 dark:bg-slate-900"
    :class="open ? 'fixed inset-y-0 right-0 z-40 flex flex-col shadow-2xl' : 'hidden md:block'"
  >
    <div class="mb-3 flex items-center justify-between">
      <h4
        class="flex items-center gap-1.5 text-xs font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
      >
        <LayoutGrid class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        Question Navigator
      </h4>
      <button
        v-if="open"
        type="button"
        class="grid h-7 w-7 place-items-center rounded-lg text-slate-400 transition hover:bg-slate-100 hover:text-slate-600 md:hidden dark:hover:bg-slate-800"
        aria-label="Close navigator"
        @click="$emit('close')"
      >
        <X class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
      </button>
    </div>

    <!-- Legend -->
    <div
      class="mb-4 grid grid-cols-2 gap-2 text-[10px] font-medium text-slate-500 dark:text-slate-400"
    >
      <div class="flex items-center gap-1.5">
        <span class="h-2.5 w-2.5 rounded-full bg-violet-600"></span>
        <span>Answered ({{ answeredCount }})</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="h-2.5 w-2.5 rounded-full bg-amber-500"></span>
        <span>Flagged ({{ flaggedCount }})</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="h-2.5 w-2.5 rounded-full bg-slate-200 dark:bg-slate-700"></span>
        <span>Unanswered ({{ total - answeredCount }})</span>
      </div>
    </div>

    <!-- Progress -->
    <div
      class="mb-3 h-1.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800"
      role="progressbar"
      :aria-valuenow="answeredCount"
      aria-valuemin="0"
      :aria-valuemax="total"
    >
      <div
        class="h-full bg-gradient-to-r from-violet-500 to-indigo-500 transition-all duration-300"
        :style="{ width: `${total ? Math.round((answeredCount / total) * 100) : 0}%` }"
      ></div>
    </div>

    <!-- Palette grid -->
    <div
      class="grid max-h-[calc(100vh-260px)] grid-cols-5 gap-2 overflow-y-auto overscroll-contain p-1"
    >
      <button
        v-for="(q, idx) in questions"
        :key="q.id"
        type="button"
        class="relative flex h-10 w-10 items-center justify-center rounded-xl text-xs font-bold transition sm:h-9 sm:w-9"
        :class="paletteItemClass(q, idx)"
        :aria-label="`Go to question ${idx + 1}`"
        :aria-current="idx === currentIndex ? 'true' : undefined"
        @click="$emit('jump', idx)"
      >
        {{ idx + 1 }}
        <span
          v-if="isFlagged(q.id)"
          class="absolute -top-1 -right-1 h-2.5 w-2.5 rounded-full bg-amber-500 ring-2 ring-white dark:ring-slate-900"
        ></span>
      </button>
    </div>

    <p class="mt-3 text-center text-[10px] font-semibold text-slate-400 tabular-nums">
      {{ answeredCount }} / {{ total }} answered
    </p>
  </aside>
</template>

<script setup lang="ts">
import { LayoutGrid, X } from '@lucide/vue';
import type { QuestionResponse } from '../../../../types';

const props = defineProps<{
  questions: QuestionResponse[];
  currentIndex: number;
  open: boolean;
  answeredCount: number;
  flaggedCount: number;
  total: number;
  isAnswered: (q: QuestionResponse) => boolean;
  isFlagged: (qId: number) => boolean;
}>();

defineEmits<{ jump: [index: number]; close: [] }>();

const paletteItemClass = (q: QuestionResponse, idx: number) => {
  if (idx === props.currentIndex) {
    return 'bg-violet-600 text-white ring-2 ring-violet-400 ring-offset-2 shadow-md dark:ring-offset-slate-900';
  }
  if (props.isAnswered(q)) {
    return 'bg-violet-100 text-violet-900 hover:bg-violet-200 dark:bg-violet-900/60 dark:text-violet-200';
  }
  if (props.isFlagged(q.id)) {
    return 'bg-amber-100 text-amber-900 hover:bg-amber-200 dark:bg-amber-900/60 dark:text-amber-200';
  }
  return 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700';
};
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
