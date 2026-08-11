<template>
  <header
    class="space-y-3 rounded-2xl border border-slate-200/80 bg-white p-3 shadow-sm sm:p-4 dark:border-slate-700/80 dark:bg-slate-800/60"
  >
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span class="relative flex h-2.5 w-2.5">
          <span
            class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"
          ></span>
          <span class="relative inline-flex h-2.5 w-2.5 rounded-full bg-emerald-500"></span>
        </span>
        <p aria-live="polite" class="font-black text-slate-800 dark:text-slate-100">
          Card {{ currentNumber }}
          <span class="text-sm font-normal text-slate-400">of {{ total }}</span>
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-1.5 sm:gap-2">
        <span
          class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-600 dark:bg-slate-700/70 dark:text-slate-300"
        >
          {{ remaining }} left
        </span>
        <span
          class="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2.5 py-1 text-xs font-semibold text-amber-700 dark:bg-amber-950/50 dark:text-amber-300"
        >
          <Clock class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          {{ dueCount }} due
        </span>
        <span
          class="inline-flex items-center gap-1 rounded-full bg-blue-100 px-2.5 py-1 text-xs font-semibold text-blue-700 dark:bg-blue-950/50 dark:text-blue-300"
        >
          <Sparkles class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          {{ newCount }} new
        </span>
        <span
          v-if="skippedCount > 0"
          class="inline-flex items-center gap-1 rounded-full bg-violet-100 px-2.5 py-1 text-xs font-semibold text-violet-700 dark:bg-violet-950/50 dark:text-violet-300"
        >
          <SkipForward class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          {{ skippedCount }} skipped
        </span>
        <Button
          size="sm"
          variant="ghost"
          class="ml-1 text-slate-500 hover:text-slate-700 dark:text-slate-400"
          @click="emit('end')"
        >
          <span class="inline-flex items-center gap-1.5">
            <LogOut class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
            End session
          </span>
        </Button>
      </div>
    </div>

    <!-- Progress bar -->
    <div
      class="h-1.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-700"
      role="progressbar"
      :aria-valuenow="progress"
      aria-valuemin="0"
      aria-valuemax="100"
    >
      <div
        class="h-full bg-gradient-to-r from-violet-500 to-indigo-500 transition-all duration-300"
        :style="{ width: `${progress}%` }"
      ></div>
    </div>

    <!-- Active filter pills -->
    <div
      v-if="allQuestions || selectedTags.length"
      class="flex scrollbar-none items-center gap-1.5 overflow-x-auto py-0.5 text-xs"
    >
      <span class="shrink-0 text-[11px] font-bold text-slate-400 uppercase">Filters:</span>
      <span
        v-if="allQuestions"
        class="shrink-0 rounded-md bg-violet-100 px-2 py-0.5 text-[11px] font-semibold text-violet-700 dark:bg-violet-900/40 dark:text-violet-300"
      >
        All questions
      </span>
      <span
        v-for="tag in selectedTags"
        :key="tag"
        class="shrink-0 rounded-md bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-600 dark:bg-slate-700 dark:text-slate-300"
      >
        {{ tag }}
      </span>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Clock, LogOut, SkipForward, Sparkles } from '@lucide/vue';
import Button from '../../ui/Button.vue';

const props = defineProps<{
  currentNumber: number;
  total: number;
  remaining: number;
  dueCount: number;
  newCount: number;
  allQuestions: boolean;
  selectedTags: string[];
  skippedCount: number;
}>();

const emit = defineEmits<{
  end: [];
}>();

const progress = computed(() => {
  if (props.total <= 0) return 0;
  return Math.min(100, Math.round(((props.currentNumber - 1) / props.total) * 100));
});
</script>

<style scoped>
.scrollbar-none {
  scrollbar-width: none;
}
.scrollbar-none::-webkit-scrollbar {
  display: none;
}
</style>
