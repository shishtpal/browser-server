<template>
  <div class="border-t border-slate-200 bg-white/80 px-4 py-3 backdrop-blur-sm dark:border-white/10 dark:bg-slate-950/80">
    <form class="mx-auto w-full lg:px-4" @submit.prevent="submit">
      <div class="relative">
        <textarea
          ref="textareaRef"
          v-model="localValue"
          class="max-h-48 min-h-[52px] w-full resize-none rounded-xl border border-slate-200 bg-white py-3 pl-4 pr-24 text-sm leading-relaxed outline-none transition-colors placeholder:text-slate-400 focus:border-slate-400 dark:border-white/10 dark:bg-slate-900 dark:placeholder:text-slate-500 dark:focus:border-white/20"
          :disabled="disabled"
          placeholder="Message the assistant…"
          rows="1"
          @input="onInput"
          @keydown.enter.exact.prevent="submit"
        ></textarea>
        <div class="absolute bottom-2 right-2 flex items-center gap-1.5">
          <button
            v-if="busy"
            class="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-xs font-bold text-slate-700 transition hover:bg-slate-100 dark:border-white/10 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700"
            type="button"
            @click="$emit('stop')"
          >
            Stop
          </button>
          <button
            class="grid h-8 w-8 place-items-center rounded-lg bg-slate-900 text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-40 dark:bg-white dark:text-slate-900 dark:hover:bg-gray-100"
            :disabled="!canSend"
            type="submit"
            title="Send message"
          >
            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19V5m0 0l-7 7m7-7l7 7"/></svg>
          </button>
        </div>
      </div>
      <p class="mt-2 text-center text-xs text-slate-400 dark:text-slate-500">
        <kbd class="rounded border border-slate-200 px-1 py-0.5 text-[10px] dark:border-white/10">Enter</kbd> to send ·
        <kbd class="rounded border border-slate-200 px-1 py-0.5 text-[10px] dark:border-white/10">Shift+Enter</kbd> for new line
      </p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  modelValue: string
  disabled: boolean
  busy: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  send: [content: string]
  stop: []
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)
const localValue = ref(props.modelValue)

// Sync parent → local only when the parent externally resets the value (e.g. suggestions)
watch(() => props.modelValue, (v) => {
  if (v !== localValue.value) localValue.value = v
})

const canSend = computed(() => !props.disabled && !props.busy && localValue.value.trim().length > 0)

function onInput() {
  const el = textareaRef.value
  if (el) {
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 192) + 'px'
  }
}

function submit() {
  if (!canSend.value) return
  const content = localValue.value.trim()
  localValue.value = ''
  emit('update:modelValue', '')
  emit('send', content)
  if (textareaRef.value) textareaRef.value.style.height = 'auto'
}

function focus() {
  textareaRef.value?.focus()
}

defineExpose({ focus })
</script>
