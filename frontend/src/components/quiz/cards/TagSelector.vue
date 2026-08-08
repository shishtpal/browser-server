<template>
  <div class="space-y-2.5">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span
          class="flex items-center gap-1.5 text-xs font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
        >
          <Tags class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Filter by Tags
        </span>
        <span
          v-if="modelValue.length && !disabled"
          class="rounded-full bg-violet-100 px-2 py-0.5 text-[10px] font-bold text-violet-700 dark:bg-violet-900/40 dark:text-violet-300"
        >
          {{ modelValue.length }} selected
        </span>
      </div>

      <div v-if="!disabled && options.length > 0" class="flex items-center gap-1.5 text-xs">
        <button
          type="button"
          class="rounded-md px-2 py-1 text-[11px] font-semibold text-violet-600 transition hover:bg-violet-50 dark:text-violet-400 dark:hover:bg-violet-950/40"
          @click="toggleAllFiltered"
        >
          {{ areAllFilteredSelected ? 'Deselect visible' : 'Select visible' }}
        </button>
        <span v-if="modelValue.length" class="text-slate-300 dark:text-slate-700">•</span>
        <button
          v-if="modelValue.length"
          type="button"
          class="rounded-md px-2 py-1 text-[11px] font-semibold text-slate-500 transition hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
          @click="clearAll"
        >
          Clear all
        </button>
      </div>
    </div>

    <!-- Search input -->
    <div v-if="options.length > 0" class="relative">
      <Search
        class="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
        aria-hidden="true"
      />
      <input
        v-model="searchQuery"
        :disabled="disabled"
        type="text"
        placeholder="Search tags..."
        class="w-full rounded-xl border border-slate-200 bg-slate-50/70 py-2 pr-8 pl-9 text-xs text-slate-800 placeholder-slate-400 transition outline-none focus:border-violet-400 focus:bg-white focus:ring-2 focus:ring-violet-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-700 dark:bg-slate-900/60 dark:text-slate-200 dark:placeholder-slate-500 dark:focus:border-violet-500 dark:focus:bg-slate-900 dark:focus:ring-violet-900/30"
      />
      <button
        v-if="searchQuery"
        type="button"
        class="absolute inset-y-0 right-0 flex items-center pr-2.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
        aria-label="Clear search"
        @click="searchQuery = ''"
      >
        <X class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
      </button>
    </div>

    <!-- Scrollable tag box -->
    <div
      class="max-h-40 scrollbar-thin scrollbar-thumb-slate-300 overflow-y-auto rounded-xl border border-slate-200 bg-slate-50/40 p-3 sm:max-h-48 dark:scrollbar-thumb-slate-600 dark:border-slate-700/80 dark:bg-slate-900/40"
      :class="{ 'pointer-events-none opacity-50': disabled }"
    >
      <p v-if="!options.length" class="py-4 text-center text-xs text-slate-500 dark:text-slate-400">
        No question tags found in your question bank.
      </p>
      <div
        v-else-if="!filteredTags.length"
        class="py-4 text-center text-xs text-slate-500 dark:text-slate-400"
      >
        No tags match "<span class="font-semibold">{{ searchQuery }}</span
        >"
        <button
          type="button"
          class="ml-1.5 text-violet-600 underline dark:text-violet-400"
          @click="searchQuery = ''"
        >
          Clear search
        </button>
      </div>
      <div v-else class="flex flex-wrap gap-1.5">
        <label
          v-for="tag in filteredTags"
          :key="tag"
          class="flex cursor-pointer items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-semibold transition-all select-none"
          :class="
            modelValue.includes(tag) && !disabled
              ? 'border-violet-300 bg-violet-100/80 text-violet-800 shadow-2xs dark:border-violet-600/70 dark:bg-violet-950/60 dark:text-violet-200'
              : 'border-slate-200/90 bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-100/60 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300 dark:hover:border-slate-600 dark:hover:bg-slate-700/50'
          "
        >
          <input
            :checked="modelValue.includes(tag)"
            :disabled="disabled"
            type="checkbox"
            :value="tag"
            class="h-3.5 w-3.5 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-slate-600 dark:bg-slate-700"
            @change="toggleTag(tag)"
          />
          <span>{{ tag }}</span>
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { Search, Tags, X } from '@lucide/vue';

const props = defineProps<{
  modelValue: string[];
  options: string[];
  disabled?: boolean;
}>();

const emit = defineEmits<{
  'update:modelValue': [tags: string[]];
}>();

const searchQuery = ref('');

const filteredTags = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  if (!query) return props.options;
  return props.options.filter((tag) => tag.toLowerCase().includes(query));
});

const areAllFilteredSelected = computed(
  () =>
    filteredTags.value.length > 0 &&
    filteredTags.value.every((tag) => props.modelValue.includes(tag)),
);

const toggleTag = (tag: string) => {
  if (props.disabled) return;
  const current = new Set(props.modelValue);
  if (current.has(tag)) current.delete(tag);
  else current.add(tag);
  emit('update:modelValue', Array.from(current));
};

const toggleAllFiltered = () => {
  if (props.disabled) return;
  if (areAllFilteredSelected.value) {
    const toRemove = new Set(filteredTags.value);
    emit(
      'update:modelValue',
      props.modelValue.filter((tag) => !toRemove.has(tag)),
    );
  } else {
    emit('update:modelValue', Array.from(new Set([...props.modelValue, ...filteredTags.value])));
  }
};

const clearAll = () => {
  if (props.disabled) return;
  emit('update:modelValue', []);
};
</script>

<style scoped>
.scrollbar-thin {
  scrollbar-width: thin;
}
</style>
