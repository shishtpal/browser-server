<template>
  <nav
    class="-mx-1 flex scrollbar-none gap-1 overflow-x-auto scroll-smooth rounded-xl bg-slate-100/80 p-1 sm:mx-0 dark:bg-slate-800/60"
    role="tablist"
    aria-label="Quiz sections"
  >
    <button
      v-for="tab in tabs"
      :key="tab.key"
      type="button"
      role="tab"
      :aria-selected="modelValue === tab.key"
      class="flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-2 text-[11px] font-black tracking-wide whitespace-nowrap transition focus:outline-none focus-visible:ring-2 focus-visible:ring-violet-400 sm:flex-1 sm:justify-center sm:px-2 sm:text-xs"
      :class="
        modelValue === tab.key
          ? 'bg-white text-violet-700 shadow-sm dark:bg-slate-900 dark:text-violet-300'
          : 'text-slate-500 hover:bg-white/60 hover:text-slate-800 dark:text-slate-400 dark:hover:bg-slate-700/60 dark:hover:text-slate-200'
      "
      @click="$emit('update:modelValue', tab.key)"
    >
      <component :is="tab.icon" class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
      <span>{{ tab.label }}</span>
      <span
        v-if="typeof tab.count === 'number'"
        class="rounded-full px-1.5 py-px text-[10px] leading-4 font-bold"
        :class="
          modelValue === tab.key
            ? 'bg-violet-100 text-violet-700 dark:bg-violet-900/50 dark:text-violet-300'
            : 'bg-slate-200/80 text-slate-500 dark:bg-slate-700 dark:text-slate-400'
        "
      >
        {{ tab.count }}
      </span>
    </button>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  FileText,
  Layers,
  LayoutDashboard,
  ListChecks,
  Sparkles,
  type LucideIcon,
} from '@lucide/vue';
import type { QuizTab } from './composables/useQuizPage';

const props = defineProps<{
  modelValue: QuizTab;
  questionCount?: number;
  paperCount?: number;
}>();

defineEmits<{ 'update:modelValue': [tab: QuizTab] }>();

const tabs = computed<Array<{ key: QuizTab; label: string; icon: LucideIcon; count?: number }>>(
  () => [
    { key: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { key: 'questions', label: 'Questions', icon: ListChecks, count: props.questionCount },
    { key: 'cards', label: 'Cards', icon: Layers },
    { key: 'generate', label: 'Generate', icon: Sparkles },
    { key: 'papers', label: 'Papers', icon: FileText, count: props.paperCount },
  ],
);
</script>

<style scoped>
.scrollbar-none {
  scrollbar-width: none;
}
.scrollbar-none::-webkit-scrollbar {
  display: none;
}
</style>
