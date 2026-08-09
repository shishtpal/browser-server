<template>
  <div v-if="allTags.length" class="space-y-2">
    <!-- Scrollable pill row -->
    <div
      class="-mx-1 flex scrollbar-none items-center gap-1 overflow-x-auto px-1 py-0.5"
      role="group"
      aria-label="Filter by tag"
    >
      <FilterPill :active="activeTag === null" class="shrink-0" @click="$emit('clear')">
        All
      </FilterPill>
      <FilterPill
        v-for="tag in allTags"
        :key="tag"
        :active="activeTag === tag"
        variant="tag"
        class="shrink-0"
        @click="$emit('select', tag)"
      >
        #{{ tag }}
      </FilterPill>
    </div>

    <!-- Active filter banner -->
    <Transition
      enter-active-class="transition-all duration-200 ease-out"
      enter-from-class="-translate-y-1 opacity-0"
      enter-to-class="translate-y-0 opacity-100"
      leave-active-class="transition-all duration-150 ease-in"
      leave-from-class="translate-y-0 opacity-100"
      leave-to-class="-translate-y-1 opacity-0"
    >
      <div
        v-if="activeTag"
        class="flex items-center gap-2 rounded-xl border border-cyan-200 bg-cyan-50/80 p-2 text-xs text-cyan-800 shadow-sm transition-colors dark:border-cyan-900/30 dark:bg-cyan-900/20 dark:text-cyan-300"
        role="status"
      >
        <TagIcon class="h-3.5 w-3.5 shrink-0" :stroke-width="2.25" aria-hidden="true" />
        <span class="font-bold">Filtering by tag:</span>
        <span
          class="rounded-md bg-white px-2 py-0.5 font-black text-cyan-700 shadow-sm transition-colors dark:bg-slate-800 dark:text-cyan-400"
        >
          {{ activeTag }}
        </span>
        <button
          type="button"
          class="ml-auto inline-flex items-center gap-1 rounded-md bg-cyan-200 px-2 py-0.5 font-black text-cyan-800 transition hover:bg-cyan-300 dark:bg-cyan-800 dark:text-cyan-200 dark:hover:bg-cyan-700"
          @click="$emit('clear')"
        >
          <X class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          Clear
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { Tag as TagIcon, X } from '@lucide/vue';
import FilterPill from '../ui/FilterPill.vue';

defineProps<{
  allTags: string[];
  activeTag: string | null;
}>();

defineEmits<{
  select: [tag: string];
  clear: [];
}>();
</script>

<style scoped>
.scrollbar-none {
  scrollbar-width: none;
}
.scrollbar-none::-webkit-scrollbar {
  display: none;
}
</style>
