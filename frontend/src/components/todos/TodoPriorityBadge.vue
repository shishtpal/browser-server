<template>
  <span
    :class="[
      'inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] font-black',
      meta.badgeClass,
    ]"
  >
    <span :class="['h-1.5 w-1.5 rounded-full', meta.dotClass]" aria-hidden="true"></span>
    {{ meta.label }}
  </span>
</template>

<script setup lang="ts">
import type { TodoPriority } from '../../types';
import { computed } from 'vue';
import { priorityMeta } from './todoFormat';

const props = defineProps<{ priority: TodoPriority }>();

const meta = computed(() => priorityMeta(props.priority));
</script>
