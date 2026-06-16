<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { isServerOnline } from '../lib/api'

const online = ref<boolean | null>(null)

let timer: number | undefined

async function check() {
  online.value = await isServerOnline()
}

onMounted(() => {
  void check()
  timer = window.setInterval(check, 30_000)
})

onBeforeUnmount(() => {
  if (timer !== undefined) window.clearInterval(timer)
})
</script>

<template>
  <div
    class="flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium transition"
    :class="online === null
      ? 'bg-slate-100 text-slate-400 dark:bg-white/5 dark:text-slate-500'
      : online
        ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400'
        : 'bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400'"
    :title="online === null ? 'Checking server…' : online ? 'Server is running' : 'Server is unreachable'"
  >
    <span
      class="inline-block h-2 w-2 rounded-full"
      :class="online === null
        ? 'animate-pulse bg-slate-300 dark:bg-slate-600'
        : online
          ? 'bg-emerald-500'
          : 'bg-rose-500'"
    />
    <span>{{ online === null ? 'Checking…' : online ? 'Server OK' : 'Offline' }}</span>
  </div>
</template>
