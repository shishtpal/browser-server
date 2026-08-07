<template>
  <Modal :open="!!paper" :title="paper?.title ?? 'Paper'" :description="description" fullscreen @close="$emit('close')">
    <div v-if="paper" class="flex h-full min-h-0 flex-col">
      <div class="mb-3 flex items-center gap-2">
        <button
          type="button"
          class="rounded-lg bg-white/10 px-3 py-1.5 text-xs font-bold text-white transition hover:bg-white/20"
          @click="showAnswers = !showAnswers"
        >
          {{ showAnswers ? 'Hide answers' : 'Show answers' }}
        </button>
      </div>
      <div class="flex-1 space-y-3 overflow-y-auto rounded-lg bg-white p-3 text-slate-900 dark:bg-slate-900 dark:text-slate-100">
        <div v-for="(q, i) in paper.questions ?? []" :key="q.id" class="rounded-lg border border-gray-200 p-3 dark:border-slate-700">
          <p class="text-sm font-semibold">
            <span class="mr-1 font-black text-violet-600 dark:text-violet-400">{{ i + 1 }}.</span>{{ q.question }}
          </p>
          <ul v-if="q.options?.length" class="mt-2 space-y-1">
            <li
              v-for="opt in q.options"
              :key="opt.index"
              class="rounded px-2 py-1 text-xs"
              :class="showAnswers && opt.correct ? 'bg-emerald-100 font-bold text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300' : 'text-slate-600 dark:text-slate-400'"
            >
              {{ String.fromCharCode(65 + opt.index) }}. {{ opt.text }}
            </li>
          </ul>
          <ol v-else-if="q.chronology_items?.length" class="mt-2 space-y-1">
            <li v-for="item in ordered(q.chronology_items)" :key="item.index" class="text-xs text-slate-600 dark:text-slate-400">
              <span v-if="showAnswers" class="mr-1 font-black text-violet-600 dark:text-violet-400">{{ item.correct_order }}.</span>{{ item.text }}
            </li>
          </ol>
          <p v-else-if="showAnswers && q.expected_text" class="mt-2 text-xs font-bold text-emerald-700 dark:text-emerald-300">
            Answer: {{ q.expected_text }}
          </p>
          <pre v-if="showAnswers && q.explanation" class="mt-2 whitespace-pre-wrap rounded bg-slate-50 p-2 text-[11px] text-slate-500 dark:bg-slate-800 dark:text-slate-400">{{ q.explanation }}</pre>
        </div>
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Modal from '../ui/Modal.vue'
import type { ChronologyItem, QuestionPaper } from '../../types'

const props = defineProps<{ paper: QuestionPaper | null }>()
defineEmits<{ close: [] }>()

const showAnswers = ref(false)
watch(() => props.paper?.id, () => (showAnswers.value = false))

const description = computed(() =>
  props.paper ? `${props.paper.question_count} questions · generated ${new Date(props.paper.created_at).toLocaleString()}` : undefined
)

const ordered = (items: ChronologyItem[]) => [...items].sort((a, b) => a.correct_order - b.correct_order)
</script>
