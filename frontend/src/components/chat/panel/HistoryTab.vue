<template>
  <div class="space-y-2 p-4">
    <template v-if="toolCalls.length > 0">
      <div
        v-for="call in toolCalls"
        :key="call.id"
        class="rounded-lg border border-slate-200 bg-white p-2.5 dark:border-white/10 dark:bg-slate-900"
      >
        <div class="flex items-center gap-2">
          <Wrench
            class="h-3 w-3 shrink-0 text-amber-500/80"
            :stroke-width="2.25"
            aria-hidden="true"
          />
          <span
            class="min-w-0 flex-1 truncate text-xs font-bold text-slate-700 dark:text-slate-300"
          >
            {{ call.name }}
          </span>
          <span
            class="shrink-0 rounded-full px-1.5 py-0.5 text-[9px] font-semibold uppercase"
            :class="statusClass(call.status)"
          >
            {{ call.status }}
          </span>
        </div>
        <details v-if="call.args" class="mt-1.5">
          <summary
            class="cursor-pointer text-[10px] font-medium text-slate-500 dark:text-slate-400"
          >
            Args
          </summary>
          <pre
            class="mt-1 max-h-24 overflow-auto rounded bg-slate-100 p-1.5 text-[10px] leading-tight dark:bg-slate-800"
            >{{ call.args }}</pre>
        </details>
        <details v-if="call.result" class="mt-1">
          <summary
            class="cursor-pointer text-[10px] font-medium text-slate-500 dark:text-slate-400"
          >
            Result
          </summary>
          <pre
            class="mt-1 max-h-24 overflow-auto rounded bg-slate-100 p-1.5 text-[10px] leading-tight dark:bg-slate-800"
            >{{ call.result }}</pre>
        </details>
      </div>
    </template>

    <!-- Empty state -->
    <div
      v-else
      class="flex flex-col items-center pt-8 text-center text-xs text-slate-400 dark:text-slate-500"
    >
      <ClipboardList class="h-8 w-8 opacity-50" :stroke-width="1.75" aria-hidden="true" />
      <p class="mt-2">No tool calls in this conversation yet.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ClipboardList, Wrench } from '@lucide/vue';
import type { ToolCallEntry } from '../messages/messageTools';

defineProps<{ toolCalls: ToolCallEntry[] }>();

function statusClass(status: string): string {
  switch (status) {
    case 'completed':
    case 'success':
      return 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400';
    case 'commented':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300';
    case 'error':
    case 'rejected':
      return 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400';
    case 'pending':
    case 'running':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300';
    default:
      return 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300';
  }
}
</script>
