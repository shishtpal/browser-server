<template>
  <div>
    <!-- Desktop table -->
    <div
      class="hidden overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm md:block dark:border-white/10 dark:bg-slate-900"
    >
      <table class="w-full text-left text-xs">
        <thead class="bg-slate-50 text-slate-500 dark:bg-white/5">
          <tr>
            <th
              v-for="heading in headings"
              :key="heading"
              class="px-3 py-2.5 font-bold tracking-wider uppercase"
            >
              {{ heading }}
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100 dark:divide-white/5">
          <tr
            v-for="log in logs"
            :key="log.id"
            tabindex="0"
            class="cursor-pointer transition hover:bg-cyan-50 focus:bg-cyan-50 focus:ring-2 focus:ring-cyan-500 focus:outline-none focus:ring-inset dark:hover:bg-cyan-950/20 dark:focus:bg-cyan-950/20"
            :class="{ 'bg-cyan-50/70 dark:bg-cyan-950/30': log.id === selectedId }"
            :aria-selected="log.id === selectedId"
            @click="$emit('select', log)"
            @keydown.enter="$emit('select', log)"
            @keydown.space.prevent="$emit('select', log)"
          >
            <td class="px-3 py-3 whitespace-nowrap">{{ formatTimestamp(log.created_at) }}</td>
            <td class="px-3 py-3">{{ sourceLabel(log.source) }}</td>
            <td class="max-w-52 truncate px-3 py-3" :title="`${log.provider} / ${log.model}`">
              <div class="flex items-center gap-1.5">
                <Bot
                  class="h-3.5 w-3.5 shrink-0 text-slate-400"
                  :stroke-width="2"
                  aria-hidden="true"
                />
                <span class="truncate">{{ log.provider }} / {{ log.model }}</span>
              </div>
            </td>
            <td class="px-3 py-3">
              <span
                class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 font-bold capitalize"
                :class="statusPillClass(log.status)"
              >
                <component
                  :is="statusIcon(log.status)"
                  class="h-3 w-3"
                  :stroke-width="2.5"
                  aria-hidden="true"
                />
                {{ log.status }}
              </span>
            </td>
            <td class="px-3 py-3 tabular-nums">{{ formatMs(log.latency_ms) }}</td>
            <td class="px-3 py-3 tabular-nums">{{ formatOptionalNumber(log.total_tokens) }}</td>
            <td class="px-3 py-3 tabular-nums">{{ log.tool_calls?.length || 0 }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Mobile cards -->
    <div class="space-y-2.5 md:hidden">
      <button
        v-for="log in logs"
        :key="log.id"
        type="button"
        class="w-full rounded-xl border border-slate-200 bg-white p-3.5 text-left shadow-sm transition focus:ring-2 focus:ring-cyan-500 focus:outline-none dark:border-white/10 dark:bg-slate-900"
        :class="{
          'border-cyan-400 ring-1 ring-cyan-400 dark:border-cyan-700 dark:ring-cyan-800':
            log.id === selectedId,
        }"
        @click="$emit('select', log)"
      >
        <div class="flex items-center justify-between gap-3">
          <span class="flex min-w-0 items-center gap-1.5">
            <Bot class="h-3.5 w-3.5 shrink-0 text-slate-400" :stroke-width="2" aria-hidden="true" />
            <strong class="truncate text-xs">{{ log.provider }} / {{ log.model }}</strong>
          </span>
          <span
            class="inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-bold capitalize"
            :class="statusPillClass(log.status)"
          >
            <component
              :is="statusIcon(log.status)"
              class="h-3 w-3"
              :stroke-width="2.5"
              aria-hidden="true"
            />
            {{ log.status }}
          </span>
        </div>
        <div
          class="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-[11px] text-slate-500 dark:text-slate-400"
        >
          <span>{{ formatTimestamp(log.created_at) }}</span>
          <span>{{ sourceLabel(log.source) }}</span>
          <span>{{ formatMs(log.latency_ms) }}</span>
          <span
            >{{ formatOptionalNumber(log.total_tokens) }} tokens ·
            {{ log.tool_calls?.length || 0 }} tools</span
          >
        </div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AIRequestLog } from '@browser-server/shared-types';
import { Ban, Bot, CircleCheck, CircleX, Clock, type LucideIcon } from '@lucide/vue';
import {
  formatMs,
  formatOptionalNumber,
  formatTimestamp,
  sourceLabel,
  statusPillClass,
} from './monitoringFormat';

defineProps<{
  logs: AIRequestLog[];
  selectedId?: string | null;
}>();

defineEmits<{ select: [log: AIRequestLog] }>();

const headings = [
  'Timestamp',
  'Source',
  'Provider / model',
  'Status',
  'Latency',
  'Tokens',
  'Tools',
];

const statusIcon = (status: string): LucideIcon => {
  switch ((status ?? '').toLowerCase()) {
    case 'success':
      return CircleCheck;
    case 'error':
      return CircleX;
    case 'cancelled':
      return Ban;
    default:
      return Clock;
  }
};
</script>
