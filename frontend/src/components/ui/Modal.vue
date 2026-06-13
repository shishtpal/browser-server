<template>
  <Teleport v-if="mounted" to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm" @click.self="close">
      <div :class="fullscreen ? 'w-full h-full flex flex-col rounded-none border-none bg-black/95 p-3 sm:p-5' : 'w-full max-w-lg rounded-xl border border-gray-200 bg-white p-5 shadow-2xl shadow-gray-900/30 dark:border-white/10 dark:bg-slate-800 dark:shadow-slate-950/30 sm:p-6'">
        <div :class="['mb-4 flex items-start justify-between gap-4', fullscreen ? 'text-white' : '']">
          <div v-if="!fullscreen" class="flex-1">
            <h2 class="text-lg font-black text-slate-900 dark:text-white">{{ title }}</h2>
            <p v-if="description" class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ description }}</p>
          </div>
          <button type="button" @click="close" :class="fullscreen ? 'ml-auto grid h-10 w-10 place-items-center rounded-lg bg-white/10 text-white text-xl hover:bg-white/20' : 'grid h-8 w-8 place-items-center rounded-lg bg-gray-100 text-slate-500 hover:bg-gray-200 dark:bg-slate-700 dark:text-slate-400 dark:hover:bg-slate-600'" aria-label="Close">&times;</button>
        </div>
        <div :class="fullscreen ? 'flex-1 min-h-0' : ''">
          <slot></slot>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface Props {
  open: boolean
  title: string
  description?: string
  fullscreen?: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  close: []
}>()

const close = () => emit('close')

const mounted = ref(false)
onMounted(() => {
  mounted.value = true
})
</script>
