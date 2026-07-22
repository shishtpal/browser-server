<template>
  <div class="space-y-1.5">
    <div v-if="args" class="overflow-hidden rounded border border-slate-200 dark:border-white/10">
      <header class="flex h-6 items-center border-b border-slate-200 bg-slate-50 px-2 dark:border-white/10 dark:bg-slate-800/60">
        <span class="text-[9px] font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Arguments</span>
      </header>
      <pre class="max-h-24 overflow-auto whitespace-pre-wrap break-words px-2 py-1.5 font-mono text-[11px] leading-snug text-slate-700 dark:text-slate-300">{{ args }}</pre>
    </div>
    <div v-if="result" class="overflow-hidden rounded border border-slate-200 dark:border-white/10">
      <header class="flex h-6 items-center border-b border-slate-200 bg-slate-50 px-2 dark:border-white/10 dark:bg-slate-800/60">
        <span class="text-[9px] font-medium uppercase tracking-wide text-slate-500 dark:text-slate-400">Result</span>
      </header>
      <pre class="max-h-32 overflow-auto whitespace-pre-wrap break-words px-2 py-1.5 font-mono text-[11px] leading-snug text-slate-700 dark:text-slate-300">{{ result }}</pre>
    </div>
    <div v-if="decision" class="text-[10px] text-slate-500 dark:text-slate-400">
      Decision: <span class="font-medium" :class="decisionClass">{{ decision }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AIMessage } from '@browser-server/shared-types'
import { computed } from 'vue'
import { parseToolContent, getToolArgs, getToolResult, getToolDecision } from './memoryUtils'

const props = defineProps<{
  message: AIMessage
}>()

const args = computed(() => getToolArgs(props.message))
const result = computed(() => getToolResult(props.message))
const decision = computed(() => getToolDecision(props.message))

const decisionClass = computed(() => {
  switch (decision.value) {
    case 'approved': return 'text-emerald-600 dark:text-emerald-400'
    case 'rejected': return 'text-red-600 dark:text-red-400'
    default: return 'text-amber-600 dark:text-amber-400'
  }
})
</script>
