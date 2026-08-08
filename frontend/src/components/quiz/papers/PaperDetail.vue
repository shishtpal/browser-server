<template>
  <Modal
    :open="!!paper"
    :title="paper?.title ?? 'Paper'"
    :description="description"
    fullscreen
    @close="$emit('close')"
  >
    <div v-if="paper" class="flex h-full min-h-0 flex-col gap-3">
      <!-- Toolbar -->
      <div
        class="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2.5 shadow-sm dark:border-slate-700 dark:bg-slate-900"
      >
        <div
          class="flex items-center gap-2 text-xs font-semibold text-slate-500 dark:text-slate-400"
        >
          <ClipboardList class="h-4 w-4 text-violet-500" :stroke-width="2.25" aria-hidden="true" />
          {{ paper.question_count }} questions
        </div>
        <Button variant="secondary" size="sm" @click="showAnswers = !showAnswers">
          <span class="inline-flex items-center gap-1.5">
            <component
              :is="showAnswers ? EyeOff : Eye"
              class="h-3.5 w-3.5"
              :stroke-width="2.25"
              aria-hidden="true"
            />
            {{ showAnswers ? 'Hide answers' : 'Show answers' }}
          </span>
        </Button>
      </div>

      <!-- Question list -->
      <div
        class="min-h-0 flex-1 space-y-3 overflow-y-auto overscroll-contain rounded-xl bg-white p-3 sm:p-4 dark:bg-slate-900"
      >
        <article
          v-for="(q, i) in paper.questions ?? []"
          :key="q.id"
          class="rounded-xl border border-gray-200 p-3 sm:p-4 dark:border-slate-700"
        >
          <div class="flex items-start gap-2">
            <span
              class="mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-lg bg-violet-100 text-[11px] font-black text-violet-700 tabular-nums dark:bg-violet-900/40 dark:text-violet-300"
            >
              {{ i + 1 }}
            </span>
            <div class="min-w-0 flex-1">
              <div class="mb-1.5 flex flex-wrap gap-1">
                <TypeBadge :type="q.type" />
                <DifficultyBadge :difficulty="q.difficulty" />
              </div>
              <p
                class="text-sm font-semibold whitespace-pre-wrap text-slate-900 dark:text-slate-100"
              >
                {{ q.question }}
              </p>
            </div>
          </div>

          <!-- Choice options -->
          <ul v-if="q.options?.length" class="mt-2 space-y-1">
            <li
              v-for="opt in q.options"
              :key="opt.index"
              class="flex items-center gap-2 rounded-md px-2 py-1 text-xs"
              :class="
                showAnswers && opt.correct
                  ? 'bg-emerald-100 font-bold text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300'
                  : 'text-slate-600 dark:text-slate-400'
              "
            >
              <span class="font-bold">{{ optionLetter(opt.index) }}.</span>
              <span class="min-w-0 flex-1">{{ opt.text }}</span>
              <CircleCheck
                v-if="showAnswers && opt.correct"
                class="h-3.5 w-3.5 shrink-0 text-emerald-600 dark:text-emerald-400"
                :stroke-width="2.5"
                aria-label="Correct answer"
              />
            </li>
          </ul>

          <!-- Chronology -->
          <ol v-else-if="q.chronology_items?.length" class="mt-2 space-y-1">
            <li
              v-for="item in orderedChronology(q.chronology_items)"
              :key="item.index"
              class="text-xs text-slate-600 dark:text-slate-400"
            >
              <span v-if="showAnswers" class="mr-1 font-black text-violet-600 dark:text-violet-400"
                >{{ item.correct_order }}.</span
              >{{ item.text }}
            </li>
          </ol>

          <!-- Free text answer -->
          <p
            v-else-if="showAnswers && q.expected_text"
            class="mt-2 text-xs font-bold text-emerald-700 dark:text-emerald-300"
          >
            Answer: {{ q.expected_text }}
          </p>

          <pre
            v-if="showAnswers && q.explanation"
            class="mt-2 rounded-lg bg-slate-50 p-2 text-[11px] whitespace-pre-wrap text-slate-500 dark:bg-slate-800 dark:text-slate-400"
            >{{ q.explanation }}</pre>
        </article>

        <p
          v-if="!paper.questions?.length"
          class="py-8 text-center text-sm text-slate-500 dark:text-slate-400"
        >
          This paper has no questions attached.
        </p>
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { ClipboardList, Eye, EyeOff, CircleCheck } from '@lucide/vue';
import type { QuestionPaper } from '../../../types';
import Button from '../../ui/Button.vue';
import Modal from '../../ui/Modal.vue';
import TypeBadge from '../ui/TypeBadge.vue';
import DifficultyBadge from '../ui/DifficultyBadge.vue';
import { formatDateTime, optionLetter, orderedChronology } from '../quizFormat';

const props = defineProps<{ paper: QuestionPaper | null }>();
defineEmits<{ close: [] }>();

const showAnswers = ref(false);
watch(
  () => props.paper?.id,
  () => (showAnswers.value = false),
);

const description = computed(() =>
  props.paper ? `Generated ${formatDateTime(props.paper.created_at)}` : undefined,
);
</script>
