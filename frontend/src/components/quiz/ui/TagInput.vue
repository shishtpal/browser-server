<template>
  <div
    class="flex min-h-[42px] flex-wrap items-center gap-1.5 rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 shadow-sm transition focus-within:border-violet-400 focus-within:ring-4 focus-within:ring-violet-100 dark:border-slate-600 dark:bg-slate-800 dark:focus-within:border-violet-500 dark:focus-within:ring-violet-900/30"
    :class="{ 'pointer-events-none opacity-60': disabled }"
  >
    <span
      v-for="tag in modelValue"
      :key="tag"
      class="flex items-center gap-1 rounded bg-violet-100 px-2 py-0.5 text-xs font-semibold text-violet-800 dark:bg-violet-900/40 dark:text-violet-200"
    >
      {{ tag }}
      <button
        type="button"
        class="-mr-0.5 flex h-4 w-4 items-center justify-center rounded-full text-violet-600 transition hover:bg-violet-200 hover:text-violet-900 dark:text-violet-300 dark:hover:bg-violet-800"
        :aria-label="`Remove tag ${tag}`"
        @click="removeTag(tag)"
      >
        <X class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
      </button>
    </span>

    <input
      v-model="draft"
      type="text"
      :list="suggestions?.length ? listId : undefined"
      :placeholder="placeholder"
      :disabled="disabled"
      class="min-w-[10ch] flex-1 border-0 bg-transparent p-0.5 text-sm font-semibold text-slate-700 placeholder:font-normal focus:outline-none disabled:cursor-not-allowed dark:text-slate-200"
      @keydown.enter.prevent="commitDraft"
      @keydown.,.prevent="commitDraft"
      @blur="commitDraft"
    />

    <datalist v-if="suggestions?.length" :id="listId">
      <option v-for="suggestion in suggestions" :key="suggestion" :value="suggestion" />
    </datalist>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { X } from '@lucide/vue';

const props = withDefaults(
  defineProps<{
    modelValue: string[];
    /** Datalist suggestions shown while typing (e.g. existing vocabulary tags). */
    suggestions?: string[];
    placeholder?: string;
    disabled?: boolean;
    /** Unique id used for the internal <datalist>; must be unique per instance. */
    listId: string;
  }>(),
  {
    suggestions: () => [],
    placeholder: 'Type tag…',
    disabled: false,
  },
);

const emit = defineEmits<{ 'update:modelValue': [tags: string[]] }>();

const draft = ref('');

/**
 * Picks up text still sitting in the input (e.g. the user typed a tag and hit
 * the form's submit button without pressing Enter first).
 */
const commitDraft = () => {
  const value = draft.value.trim();
  if (!value) return;
  if (!props.modelValue.includes(value)) {
    emit('update:modelValue', [...props.modelValue, value]);
  }
  draft.value = '';
};

const removeTag = (tag: string) => {
  emit(
    'update:modelValue',
    props.modelValue.filter((t) => t !== tag),
  );
};

/** Forces any in-progress draft into the model; called by parents on submit. */
const flush = () => commitDraft();

defineExpose({ flush });
</script>
