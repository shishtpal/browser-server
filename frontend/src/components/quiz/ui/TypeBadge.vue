<template>
  <span
    class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[10px] font-black tracking-wide uppercase"
    :class="meta.badgeClass"
  >
    <component :is="meta.icon" class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
    {{ meta.label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { QuestionType } from '../../../types';
import { QUESTION_TYPE_META } from '../quizFormat';

const props = defineProps<{ type: QuestionType }>();

const meta = computed(() => QUESTION_TYPE_META[props.type] ?? QUESTION_TYPE_META.single_choice);
</script>
