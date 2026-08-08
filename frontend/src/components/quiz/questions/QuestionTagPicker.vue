<template>
  <div ref="root" class="relative">
    <button
      type="button"
      class="flex h-full w-full items-center justify-center gap-1.5 rounded-lg border px-2.5 py-2 text-xs font-semibold transition sm:justify-start"
      :class="
        isOpen || modelValue.length
          ? 'border-violet-400 bg-violet-50 text-violet-700 dark:border-violet-500/60 dark:bg-violet-950/40 dark:text-violet-300'
          : 'border-gray-300 bg-white text-slate-700 hover:border-slate-400 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-slate-500'
      "
      :aria-expanded="isOpen"
      @click="isOpen = !isOpen"
    >
      <Tags class="h-3.5 w-3.5 shrink-0" :stroke-width="2.25" aria-hidden="true" />
      <span>Tags</span>
      <span
        v-if="modelValue.length"
        class="rounded-full bg-violet-600 px-1.5 py-px text-[10px] leading-4 font-bold text-white"
      >
        {{ modelValue.length }}
      </span>
      <ChevronDown
        class="ml-auto h-3.5 w-3.5 shrink-0 opacity-60 transition-transform sm:ml-0"
        :class="{ 'rotate-180': isOpen }"
        aria-hidden="true"
      />
    </button>

    <Transition name="popover">
      <div
        v-if="isOpen"
        class="absolute inset-x-0 z-20 mt-1.5 w-full min-w-64 rounded-xl border border-gray-200 bg-white p-3 shadow-xl sm:right-auto sm:min-w-72 dark:border-slate-600 dark:bg-slate-800"
      >
        <!-- Selected chips -->
        <div v-if="modelValue.length" class="mb-2 flex flex-wrap gap-1">
          <span
            v-for="tag in modelValue"
            :key="tag"
            class="flex items-center gap-1 rounded bg-violet-100 px-2 py-0.5 text-[10px] font-semibold text-violet-800 dark:bg-violet-900/40 dark:text-violet-200"
          >
            {{ tag }}
            <button
              type="button"
              class="text-violet-600 hover:text-violet-900 dark:text-violet-300"
              :aria-label="`Remove tag ${tag}`"
              @click.stop="toggleTag(tag)"
            >
              <X class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
            </button>
          </span>
        </div>

        <!-- Free-form add -->
        <div class="mb-2 flex gap-1.5">
          <input
            v-model="draft"
            type="text"
            placeholder="Add tag…"
            class="min-w-0 flex-1 rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-xs focus:border-violet-400 focus:ring-2 focus:ring-violet-100 focus:outline-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-200 dark:focus:ring-violet-900/30"
            @keydown.enter.prevent="addDraft"
            @keydown.,.prevent="addDraft"
          />
          <button
            type="button"
            class="grid h-[30px] w-[30px] shrink-0 place-items-center rounded-lg bg-slate-900 text-white transition hover:bg-slate-700 dark:bg-white dark:text-slate-900 dark:hover:bg-slate-200"
            aria-label="Add tag"
            @click="addDraft"
          >
            <Plus class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          </button>
        </div>

        <!-- Available tags -->
        <div v-if="availableTags.length" class="max-h-40 space-y-0.5 overflow-y-auto">
          <label
            v-for="tag in availableTags"
            :key="tag"
            class="flex cursor-pointer items-center gap-2 rounded-lg px-1.5 py-1 text-xs hover:bg-slate-100 dark:hover:bg-slate-700"
          >
            <input
              type="checkbox"
              :checked="modelValue.includes(tag)"
              class="h-3.5 w-3.5 rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-slate-600 dark:bg-slate-700"
              @change="toggleTag(tag)"
            />
            <span class="truncate text-slate-700 dark:text-slate-200">{{ tag }}</span>
          </label>
        </div>
        <p v-else class="py-1 text-[10px] text-slate-400">No tags yet. Add some above.</p>

        <div class="mt-2 flex gap-1.5">
          <button
            v-if="modelValue.length"
            type="button"
            class="flex-1 rounded-lg bg-white px-2 py-1.5 text-xs font-bold text-slate-600 ring-1 ring-gray-300 transition ring-inset hover:bg-gray-50 dark:bg-slate-800 dark:text-slate-300 dark:ring-slate-600 dark:hover:bg-slate-700"
            @click="$emit('update:modelValue', [])"
          >
            Clear
          </button>
          <button
            type="button"
            class="flex-1 rounded-lg bg-slate-900 px-2 py-1.5 text-xs font-bold text-white transition hover:bg-slate-700 dark:bg-white dark:text-slate-900 dark:hover:bg-slate-200"
            @click="applyAndClose"
          >
            Apply
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { onClickOutside } from '@vueuse/core';
import { ChevronDown, Plus, Tags, X } from '@lucide/vue';

const props = defineProps<{
  modelValue: string[];
  availableTags: string[];
}>();

const emit = defineEmits<{
  'update:modelValue': [tags: string[]];
  apply: [];
}>();

const root = ref<HTMLElement | null>(null);
const isOpen = ref(false);
const draft = ref('');

onClickOutside(root, () => (isOpen.value = false));

function toggleTag(tag: string) {
  const next = props.modelValue.includes(tag)
    ? props.modelValue.filter((t) => t !== tag)
    : [...props.modelValue, tag];
  emit('update:modelValue', next);
}

function addDraft() {
  const value = draft.value.trim();
  if (!value) return;
  if (!props.modelValue.includes(value)) {
    emit('update:modelValue', [...props.modelValue, value]);
  }
  draft.value = '';
}

function applyAndClose() {
  addDraft();
  emit('apply');
  isOpen.value = false;
}
</script>

<style scoped>
.popover-enter-active,
.popover-leave-active {
  transition:
    opacity 0.15s ease,
    transform 0.15s ease;
}
.popover-enter-from,
.popover-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
