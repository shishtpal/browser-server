<template>
  <div
    v-if="visible"
    class="absolute inset-x-0 bottom-full z-50 mb-2 max-h-80 overflow-auto rounded-xl border border-slate-200 bg-white shadow-xl dark:border-white/10 dark:bg-slate-900"
  >
    <div v-if="loading" class="p-3 text-center text-xs text-slate-400">Searching…</div>
    <template v-else-if="grouped.length">
      <template v-for="group in grouped" :key="group.folderName || '__unfiled__'">
        <div v-if="group.folderName" class="px-3 pt-2 pb-1 text-[0.7em] font-semibold uppercase tracking-wide text-slate-400">
          {{ group.folderName }}
        </div>
        <button
          v-for="(prompt, idx) in group.items"
          :key="prompt.id"
          :class="[
            'flex w-full flex-col gap-0.5 px-3 py-2 text-left',
            activeIndex === globalIndex(group, idx) ? 'bg-slate-50 dark:bg-white/10' : 'hover:bg-slate-50 dark:hover:bg-white/5',
          ]"
          type="button"
          @click="choose(prompt)"
        >
          <span class="text-sm font-semibold text-slate-900 dark:text-white">{{ prompt.title }}</span>
          <span class="line-clamp-1 text-xs text-slate-500 dark:text-slate-400">{{ prompt.description || prompt.content }}</span>
        </button>
      </template>
    </template>
    <div v-else class="p-3 text-center text-xs text-slate-400">No matching prompts</div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, type PropType } from 'vue'
import type { PromptResponse } from '../../types'

const props = defineProps({
  visible: { type: Boolean, default: false },
  results: { type: Array as PropType<PromptResponse[]>, default: () => [] },
  loading: { type: Boolean, default: false },
})

const emit = defineEmits<{
  select: [prompt: PromptResponse]
  close: []
}>()

const activeIndex = ref(0)

const grouped = computed(() => {
  const map = new Map<string, PromptResponse[]>()
  for (const p of props.results) {
    const key = p.folder_name || '__unfiled__'
    const list = map.get(key) || []
    list.push(p)
    map.set(key, list)
  }
  return Array.from(map.entries()).map(([folderName, items]) => ({ folderName: folderName === '__unfiled__' ? '' : folderName, items }))
})

const flatLength = computed(() => props.results.length)

function globalIndex(group: { folderName: string; items: PromptResponse[] }, idx: number) {
  let acc = 0
  for (const g of grouped.value) {
    if (g === group) return acc + idx
    acc += g.items.length
  }
  return acc
}

watch(() => props.visible, (val) => {
  if (val) activeIndex.value = 0
})

watch(() => props.results.length, (len) => {
  if (len) activeIndex.value = 0
})

function choose(prompt: PromptResponse) {
  emit('select', prompt)
  emit('close')
}

function move(delta: number) {
  if (!flatLength.value) return
  activeIndex.value = (activeIndex.value + delta + flatLength.value) % flatLength.value
}

function activate() {
  if (props.results[activeIndex.value]) choose(props.results[activeIndex.value])
}

function close() {
  emit('close')
}

defineExpose({ move, activate, close })
</script>
