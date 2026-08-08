<template>
  <div
    class="overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-sm dark:border-slate-700/80 dark:bg-slate-800/60"
  >
    <!-- Header -->
    <div
      class="border-b border-slate-100 bg-slate-50/50 px-4 py-4 sm:px-5 dark:border-slate-700/50 dark:bg-slate-900/30"
    >
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-2.5">
          <div
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-violet-100 text-violet-600 shadow-2xs dark:bg-violet-900/40 dark:text-violet-300"
          >
            <Layers class="h-5 w-5" :stroke-width="2.25" aria-hidden="true" />
          </div>
          <div>
            <h2 class="text-base font-black text-slate-800 sm:text-lg dark:text-slate-100">
              Review Flashcards
            </h2>
            <p class="text-xs text-slate-500 dark:text-slate-400">
              Choose tags to study or practice all cards with spaced repetition.
            </p>
          </div>
        </div>

        <div
          v-if="nothingDue"
          class="flex items-center gap-1.5 rounded-full border border-amber-200/80 bg-amber-50 px-3 py-1 text-xs font-semibold text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-300"
        >
          <Info class="h-3.5 w-3.5 shrink-0" :stroke-width="2.25" aria-hidden="true" />
          <span>Nothing due for this selection</span>
        </div>
      </div>
    </div>

    <div class="space-y-5 p-4 sm:p-5">
      <!-- Mode & session size -->
      <div class="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
        <label
          class="relative flex cursor-pointer items-center justify-between gap-3 rounded-xl border p-3.5 transition-all select-none"
          :class="
            allQuestions
              ? 'border-violet-400 bg-violet-50/60 ring-2 ring-violet-200 dark:border-violet-500 dark:bg-violet-950/30 dark:ring-violet-900/40'
              : 'border-slate-200 bg-white hover:border-slate-300 dark:border-slate-700 dark:bg-slate-900/40 dark:hover:border-slate-600'
          "
        >
          <div class="flex items-center gap-3">
            <span
              class="grid h-8 w-8 shrink-0 place-items-center rounded-lg"
              :class="
                allQuestions
                  ? 'bg-violet-600 text-white'
                  : 'bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400'
              "
            >
              <InfinityIcon class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
            </span>
            <div>
              <span class="block text-xs font-bold text-slate-800 dark:text-slate-200">
                All Questions
              </span>
              <span class="block text-[11px] text-slate-500 dark:text-slate-400">
                Ignore tag filters and review every available card
              </span>
            </div>
          </div>
          <input
            :checked="allQuestions"
            type="checkbox"
            class="h-4 w-4 shrink-0 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-slate-600 dark:bg-slate-800"
            @change="emit('update:allQuestions', ($event.target as HTMLInputElement).checked)"
          />
        </label>

        <div
          class="flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white p-3.5 dark:border-slate-700 dark:bg-slate-900/40"
        >
          <div class="flex items-center gap-3">
            <span
              class="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400"
            >
              <ListOrdered class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
            </span>
            <div>
              <span class="block text-xs font-bold text-slate-800 dark:text-slate-200">
                Session Size
              </span>
              <span class="block text-[11px] text-slate-500 dark:text-slate-400">
                Maximum cards per review batch
              </span>
            </div>
          </div>
          <select
            :value="limit"
            class="rounded-lg border border-slate-300 bg-slate-50 px-3 py-1.5 text-xs font-bold text-slate-700 transition outline-none focus:border-violet-500 focus:ring-2 focus:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-violet-900/30"
            @change="emit('update:limit', Number(($event.target as HTMLSelectElement).value))"
          >
            <option :value="10">10 cards</option>
            <option :value="20">20 cards</option>
            <option :value="50">50 cards</option>
          </select>
        </div>
      </div>

      <!-- Tags -->
      <TagSelector v-model="selectedTagsProxy" :options="tagOptions" :disabled="allQuestions" />

      <!-- Actions -->
      <div class="flex flex-col gap-3 pt-2 sm:flex-row sm:items-center">
        <Button
          variant="gradient-violet"
          size="md"
          class="w-full sm:w-auto"
          :disabled="!canStart"
          @click="emit('start')"
        >
          <span class="inline-flex items-center gap-1.5">
            <Play class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
            Start review session
          </span>
        </Button>

        <Button
          v-if="nothingDue"
          variant="secondary"
          size="md"
          class="w-full sm:w-auto"
          @click="emit('start-practice')"
        >
          <span class="inline-flex items-center gap-1.5">
            <Dumbbell class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
            Practice these cards anyway
          </span>
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Dumbbell, Infinity as InfinityIcon, Info, Layers, ListOrdered, Play } from '@lucide/vue';
import Button from '../../ui/Button.vue';
import TagSelector from './TagSelector.vue';

const props = defineProps<{
  allQuestions: boolean;
  limit: number;
  selectedTags: string[];
  tagOptions: string[];
  canStart: boolean;
  nothingDue: boolean;
}>();

const emit = defineEmits<{
  'update:allQuestions': [value: boolean];
  'update:limit': [value: number];
  'update:selectedTags': [value: string[]];
  start: [];
  'start-practice': [];
}>();

const selectedTagsProxy = computed({
  get: () => props.selectedTags,
  set: (value: string[]) => emit('update:selectedTags', value),
});
</script>
