<template>
  <span
    class="rounded-md px-2 py-0.5 text-[10px] font-bold tracking-wide uppercase"
    :class="meta.badgeClass"
  >
    {{ meta.label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { QuestionDifficulty } from '../../../types';
import { DIFFICULTY_META } from '../quizFormat';

const props = defineProps<{ difficulty: QuestionDifficulty }>();

const meta = computed(() => DIFFICULTY_META[props.difficulty] ?? DIFFICULTY_META.medium);
</script>
