<template>
  <div class="chat-scroll flex-1 overflow-auto bg-slate-50/40 px-4 py-4 dark:bg-slate-950/40">
    <!-- Loading skeletons -->
    <div v-if="loading" class="grid gap-4" :class="gridClass">
      <div
        v-for="n in 6"
        :key="n"
        class="h-40 animate-pulse rounded-xl border border-slate-200 bg-white dark:border-white/5 dark:bg-white/5"
      ></div>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="prompts.length === 0"
      class="flex h-full flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-slate-300 bg-white/50 p-10 text-center dark:border-white/10 dark:bg-white/5"
    >
      <div
        class="grid h-14 w-14 place-items-center rounded-2xl bg-gradient-to-br from-indigo-500 to-violet-600 text-white shadow-lg shadow-indigo-500/25"
      >
        <svg
          class="h-7 w-7"
          fill="none"
          stroke="currentColor"
          stroke-width="1.6"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M12 3v3m0 12v3M5.6 5.6l2.1 2.1m8.6 8.6 2.1 2.1M3 12h3m12 0h3M5.6 18.4l2.1-2.1m8.6-8.6 2.1-2.1"
          />
        </svg>
      </div>
      <p class="text-[0.85rem] font-medium text-slate-600 dark:text-slate-300">
        {{ search ? 'No prompts match your search.' : 'No prompts match this tag yet.' }}
      </p>
      <button
        class="rounded-lg bg-indigo-600 px-3.5 py-2 text-[0.8rem] font-semibold text-white shadow-sm transition hover:bg-indigo-700"
        type="button"
        @click="$emit('create')"
      >
        Create your first prompt
      </button>
    </div>

    <!-- Cards -->
    <div v-else class="grid gap-4" :class="gridClass">
      <PromptCard
        v-for="prompt in prompts"
        :key="prompt.id"
        :prompt="prompt"
        :copied="copiedId === prompt.id"
        :dense="layout === 'list'"
        @open="$emit('open', $event)"
        @use="$emit('use', $event)"
        @copy="$emit('copy', $event)"
        @delete="$emit('delete', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PromptResponse } from '../../types'
import type { PromptLayout } from '../../composables/usePromptManager'
import { computed } from 'vue'
import PromptCard from './PromptCard.vue'

const props = defineProps<{
  prompts: PromptResponse[]
  loading: boolean
  layout: PromptLayout
  search: string
  copiedId: number | null
}>()

defineEmits<{
  open: [prompt: PromptResponse]
  use: [prompt: PromptResponse]
  copy: [prompt: PromptResponse]
  delete: [prompt: PromptResponse]
  create: []
}>()

const gridClass = computed(() =>
  props.layout === 'list'
    ? 'grid-cols-1'
    : 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4',
)
</script>
