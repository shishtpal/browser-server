<template>
  <div
    v-if="totalPages > 1"
    class="flex flex-col items-center gap-2 border-t border-gray-200 pt-4 sm:flex-row sm:justify-between dark:border-slate-700"
  >
    <p class="order-2 text-xs text-slate-500 sm:order-1 dark:text-slate-400">
      Showing <span class="font-bold">{{ startItem }}</span> to
      <span class="font-bold">{{ endItem }}</span> of
      <span class="font-bold">{{ total }}</span> results
    </p>

    <div class="order-1 flex w-full items-center gap-2 sm:order-2 sm:w-auto">
      <button
        type="button"
        :disabled="page === 1"
        class="inline-flex flex-1 items-center justify-center gap-1 rounded-lg bg-white px-3 py-2 text-xs font-semibold text-slate-700 shadow-sm ring-1 ring-gray-300 transition ring-inset hover:bg-gray-50 disabled:opacity-50 sm:flex-none dark:bg-slate-800 dark:text-slate-200 dark:ring-slate-600 dark:hover:bg-slate-700"
        @click="goTo(page - 1)"
      >
        <ChevronLeft class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        Previous
      </button>
      <div
        class="shrink-0 px-2 text-xs font-semibold text-slate-700 tabular-nums dark:text-slate-200"
      >
        {{ page }} / {{ totalPages }}
      </div>
      <button
        type="button"
        :disabled="page === totalPages"
        class="inline-flex flex-1 items-center justify-center gap-1 rounded-lg bg-white px-3 py-2 text-xs font-semibold text-slate-700 shadow-sm ring-1 ring-gray-300 transition ring-inset hover:bg-gray-50 disabled:opacity-50 sm:flex-none dark:bg-slate-800 dark:text-slate-200 dark:ring-slate-600 dark:hover:bg-slate-700"
        @click="goTo(page + 1)"
      >
        Next
        <ChevronRight class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ChevronLeft, ChevronRight } from '@lucide/vue';

const props = defineProps<{
  page: number;
  total: number;
  perPage: number;
}>();

const emit = defineEmits<{ 'update:page': [page: number] }>();

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.perPage)));
const startItem = computed(() => (props.total === 0 ? 0 : (props.page - 1) * props.perPage + 1));
const endItem = computed(() => Math.min(props.page * props.perPage, props.total));

const goTo = (page: number) => {
  if (page < 1 || page > totalPages.value || page === props.page) return;
  emit('update:page', page);
};
</script>
