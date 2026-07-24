<template>
  <div>
    <div class="flex flex-wrap items-center gap-1">
      <button
        v-for="tag in tags"
        :key="tag"
        type="button"
        @click="$emit('remove-tag', tag)"
        class="inline-flex items-center gap-1 rounded-md bg-cyan-50 px-2 py-0.5 text-[10px] font-black text-cyan-700 transition hover:bg-cyan-100 dark:bg-cyan-900/30 dark:text-cyan-400 dark:hover:bg-cyan-900/50"
      >
        {{ tag }}
        <span class="text-[8px] leading-none">✕</span>
      </button>
      <input
        v-if="showInput"
        ref="inputRef"
        :value="newTag"
        @input="newTag = ($event.target as HTMLInputElement).value"
        @keydown.enter.prevent="commitTag"
        @keydown.backspace="onBackspace"
        @blur="commitTag"
        placeholder="+ tag"
        class="w-20 rounded-md border border-gray-300 bg-white px-1.5 py-0.5 text-[10px] font-black text-slate-700 focus:border-cyan-400 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
      />
      <button
        v-else
        type="button"
        @click="openInput"
        class="rounded-md px-2 py-0.5 text-[10px] font-black text-slate-400 transition hover:bg-gray-100 dark:text-slate-500 dark:hover:bg-slate-700"
      >
        + tag
      </button>
    </div>
    <div v-if="suggestions.length" class="mt-1 flex flex-wrap gap-1">
      <button
        v-for="s in availableSuggestions"
        :key="s"
        type="button"
        @click="$emit('add-suggestion', s)"
        class="rounded-full bg-gray-100 px-2 py-0.5 text-[10px] font-bold text-slate-500 transition hover:bg-cyan-50 hover:text-cyan-700 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-cyan-900/30 dark:hover:text-cyan-400"
      >
        {{ s }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, type PropType } from 'vue'

const props = defineProps({
  modelValue: { type: Array as PropType<string[]>, default: () => [] },
  suggestions: { type: Array as PropType<string[]>, default: () => [] },
})

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
  'remove-tag': [tag: string]
  'add-suggestion': [tag: string]
}>()

const showInput = ref(false)
const newTag = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

const tags = computed(() => props.modelValue)

const availableSuggestions = computed(() => {
  if (!newTag.value) {
    return props.suggestions.filter(s => !props.modelValue.includes(s)).slice(0, 6)
  }
  return props.suggestions
    .filter(s => s.toLowerCase().includes(newTag.value.toLowerCase()) && !props.modelValue.includes(s))
    .slice(0, 6)
})

function openInput() {
  showInput.value = true
  nextTick(() => inputRef.value?.focus())
}

function commitTag() {
  const trimmed = newTag.value.trim().toLowerCase()
  if (trimmed && !props.modelValue.includes(trimmed)) {
    emit('update:modelValue', [...props.modelValue, trimmed])
  }
  newTag.value = ''
  showInput.value = false
}

function onBackspace() {
  if (newTag.value === '' && props.modelValue.length > 0) {
    emit('update:modelValue', props.modelValue.slice(0, -1))
  }
}
</script>
