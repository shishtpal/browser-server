<script setup lang="ts">
import type { PanelStatus } from './types'
import type { DatePreset, GroupBy } from '../composables/useAnalyticsView'
import { computed, watch } from 'vue'
import { faviconUrl, formatDuration } from '@browser-server/shared-utils'
import {
  createApiClient,
  useAnalyticsView,
  useExtensionSettings,
  useUserId,
} from '../composables/composables'

const emit = defineEmits<{ (event: 'status', status: PanelStatus): void }>()

const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const {
  summary,
  datePreset,
  customStart,
  customEnd,
  groupBy,
  isLoading,
  errorMessage,
  dateRange,
  timelineLabels,
  load,
} = useAnalyticsView(client, userId)

defineExpose({ refresh: load })

const presets: { value: DatePreset; label: string }[] = [
  { value: 'today', label: 'Today' },
  { value: '7days', label: '7 Days' },
  { value: '30days', label: '30 Days' },
  { value: 'custom', label: 'Custom' },
]

const groupOptions: { value: GroupBy; label: string }[] = [
  { value: 'day', label: 'Day' },
  { value: 'week', label: 'Week' },
  { value: 'month', label: 'Month' },
]

const isReady = computed(() => Boolean(client.value) && userId.value > 0)
const showSkeleton = computed(() => isLoading.value && !summary.value)

const maxTimelineValue = computed(() => {
  if (!summary.value?.timeline.length) return 0
  return Math.max(...summary.value.timeline.map((t) => t.total_seconds))
})

const chartMaxHeight = 120

function barColor(index: number, total: number): string {
  const colors = [
    'bg-rose-500',
    'bg-rose-400',
    'bg-rose-300',
    'bg-rose-200',
    'bg-slate-400',
    'bg-slate-500',
    'bg-slate-600',
    'bg-slate-700',
    'bg-slate-800',
    'bg-slate-900',
  ]
  return colors[index % colors.length]
}

watch(
  [isReady, isLoading, errorMessage, summary],
  () => {
    emit('status', {
      count: summary.value?.domains?.length ?? 0,
      state: errorMessage.value ? 'error' : isLoading.value ? 'loading' : 'ready',
    })
  },
  { immediate: true, deep: true },
)

watch(
  [client, userId, datePreset, groupBy, customStart, customEnd],
  () => {
    if (isReady.value) void load()
  },
  { immediate: true },
)
</script>

<template>
  <section class="flex flex-col">
    <!-- Controls -->
    <div class="space-y-2 border-b border-slate-800 p-3">
      <!-- Date preset pills -->
      <div class="flex items-center gap-1">
        <button
          v-for="preset in presets"
          :key="preset.value"
          type="button"
          class="rounded-md px-2.5 py-1 text-[11px] font-medium transition"
          :class="datePreset === preset.value
            ? 'bg-rose-500/15 text-rose-300 ring-1 ring-inset ring-rose-500/30'
            : 'text-slate-400 hover:bg-slate-800 hover:text-slate-300'"
          @click="datePreset = preset.value"
        >
          {{ preset.label }}
        </button>
      </div>

      <!-- Custom date inputs -->
      <div v-if="datePreset === 'custom'" class="flex items-center gap-2">
        <input
          v-model="customStart"
          type="date"
          class="w-full rounded-md border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-200 outline-none focus:border-rose-400"
        />
        <span class="text-xs text-slate-500">to</span>
        <input
          v-model="customEnd"
          type="date"
          class="w-full rounded-md border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-200 outline-none focus:border-rose-400"
        />
      </div>

      <!-- Group by toggle (only show when range > 1 day) -->
      <div v-if="datePreset !== 'today'" class="flex items-center gap-1">
        <span class="text-[11px] text-slate-500">Group by</span>
        <button
          v-for="opt in groupOptions"
          :key="opt.value"
          type="button"
          class="rounded-md px-2 py-0.5 text-[10px] font-medium transition"
          :class="groupBy === opt.value
            ? 'bg-rose-500/15 text-rose-300 ring-1 ring-inset ring-rose-500/30'
            : 'text-slate-400 hover:bg-slate-800 hover:text-slate-300'"
          @click="groupBy = opt.value"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>

    <div class="px-3 py-3">
      <!-- Loading skeleton -->
      <div v-if="showSkeleton" class="space-y-3">
        <div class="h-8 w-24 animate-pulse rounded bg-slate-800" />
        <div class="space-y-2">
          <div v-for="n in 5" :key="n" class="flex items-center gap-2">
            <div class="h-3 w-3 shrink-0 animate-pulse rounded bg-slate-800" />
            <div class="h-4 animate-pulse rounded bg-slate-800" :style="{ width: `${80 - n * 10}%` }" />
          </div>
        </div>
      </div>

      <!-- Error -->
      <div v-else-if="errorMessage" class="flex flex-col items-center gap-3 py-12 text-center">
        <div class="flex h-12 w-12 items-center justify-center rounded-full bg-rose-500/10">
          <svg class="h-6 w-6 text-rose-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 9v4M12 17h.01" />
            <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          </svg>
        </div>
        <p class="text-sm font-medium text-slate-300">Can't reach the server</p>
        <p class="max-w-[280px] text-xs text-slate-500">{{ errorMessage }}</p>
        <button
          type="button"
          class="rounded-lg bg-rose-500 px-4 py-1.5 text-xs font-medium text-white transition hover:bg-rose-400"
          @click="load"
        >
          Try again
        </button>
      </div>

      <!-- No data -->
      <div v-else-if="!summary || summary.total_seconds === 0" class="flex flex-col items-center gap-3 py-12 text-center">
        <div class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-800">
          <svg class="h-6 w-6 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 20V10" />
            <path d="M18 20V4" />
            <path d="M6 20v-4" />
          </svg>
        </div>
        <p class="text-sm font-medium text-slate-300">No activity tracked</p>
        <p class="max-w-[240px] text-xs text-slate-500">Your browsing time will appear here as you use the web.</p>
      </div>

      <!-- Data -->
      <template v-else>
        <!-- Total time -->
        <div class="mb-4 text-center">
          <div class="text-2xl font-bold text-white tabular-nums">
            {{ formatDuration(summary.total_seconds) }}
          </div>
          <div class="text-[11px] text-slate-500">total {{ datePreset === 'today' ? 'today' : `· ${dateRange.start} to ${dateRange.end}` }}</div>
        </div>

        <!-- Domain breakdown -->
        <div class="mb-4">
          <h3 class="mb-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500">Top Domains</h3>
          <ul class="space-y-1.5">
            <li
              v-for="(domain, index) in summary.domains"
              :key="domain.domain"
              class="group flex items-center gap-2"
            >
              <img
                :src="faviconUrl(`https://${domain.domain}`)"
                alt=""
                class="h-3.5 w-3.5 shrink-0 rounded-sm"
                @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
              />
              <span class="w-24 shrink-0 truncate text-xs text-slate-300" :title="domain.domain">
                {{ domain.domain }}
              </span>
              <div class="flex h-3.5 flex-1 overflow-hidden rounded-sm bg-slate-800">
                <div
                  class="h-full rounded-sm transition-all"
                  :class="barColor(index, summary.domains.length)"
                  :style="{ width: `${Math.max(domain.percentage, 2)}%` }"
                />
              </div>
              <span class="w-12 text-right text-[11px] tabular-nums text-slate-400">
                {{ formatDuration(domain.total_seconds) }}
              </span>
              <span class="w-10 text-right text-[10px] tabular-nums text-slate-500">
                {{ domain.percentage }}%
              </span>
            </li>
          </ul>
        </div>

        <!-- Trend chart -->
        <div v-if="summary.timeline.length > 0" class="mb-2">
          <h3 class="mb-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500">
            Trend · {{ groupBy }}
          </h3>
          <div class="flex items-end justify-between gap-1" :style="{ height: `${chartMaxHeight + 20}px` }">
            <div
              v-for="(point, index) in summary.timeline"
              :key="point.period"
              class="flex flex-1 flex-col items-center justify-end"
              :title="formatDuration(point.total_seconds)"
            >
              <span class="mb-1 text-[9px] tabular-nums text-slate-400">
                {{ formatDuration(point.total_seconds) }}
              </span>
              <div
                class="w-full rounded-t-sm bg-rose-500/60 transition hover:bg-rose-400/80"
                :style="{ height: maxTimelineValue > 0 ? `${(point.total_seconds / maxTimelineValue) * chartMaxHeight}px` : '0px' }"
              />
              <span
                class="mt-1 text-[8px] leading-tight text-slate-500"
                :class="{ 'opacity-0': point.total_seconds === 0 }"
              >
                {{ timelineLabels[index] }}
              </span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </section>
</template>
