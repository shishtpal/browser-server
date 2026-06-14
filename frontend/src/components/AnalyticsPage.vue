<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Domain time" title="Usage" color="rose">
      <template #stats>
        <StatCard :value="totalDuration" label="Duration" variant="primary" color="rose" />
        <StatCard :value="domainCount" label="Domains" variant="dark" color="rose" />
      </template>
      <template #actions>
        <UserSelector id="analytics-user" v-model="selectedUserId" :users="users" color="rose" />
      </template>
    </PageHeader>

    <SelectUserPrompt title="Select a user to view usage analytics" :users-count="users.length" :selected-user-id="selectedUserId" />

    <LoadingSpinner v-if="isLoading" message="Loading usage data..." color="rose" />

    <ErrorBanner v-else-if="error" :message="error" :on-retry="load" />

    <div v-else-if="selectedUserId">
      <!-- Date controls -->
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <button
          v-for="preset in presets"
          :key="preset.value"
          type="button"
          class="rounded-lg px-3 py-1.5 text-xs font-semibold transition"
          :class="datePreset === preset.value
            ? 'bg-rose-100 text-rose-700 ring-1 ring-rose-300 dark:bg-rose-900/30 dark:text-rose-300 dark:ring-rose-700'
            : 'bg-gray-100 text-slate-600 hover:bg-gray-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'"
          @click="datePreset = preset.value"
        >
          {{ preset.label }}
        </button>

        <template v-if="datePreset === 'custom'">
          <input
            v-model="customStart"
            type="date"
            class="rounded-lg border border-gray-200 bg-white px-2.5 py-1.5 text-xs text-slate-700 outline-none focus:border-rose-400 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
          />
          <span class="text-xs text-slate-500">to</span>
          <input
            v-model="customEnd"
            type="date"
            class="rounded-lg border border-gray-200 bg-white px-2.5 py-1.5 text-xs text-slate-700 outline-none focus:border-rose-400 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
          />
        </template>

        <div v-if="datePreset !== 'today'" class="ml-auto flex items-center gap-2">
          <span class="text-[11px] font-medium text-slate-500">Group by</span>
          <button
            v-for="opt in groupOptions"
            :key="opt.value"
            type="button"
            class="rounded-md px-2.5 py-1 text-[11px] font-semibold transition"
            :class="groupBy === opt.value
              ? 'bg-rose-100 text-rose-700 ring-1 ring-rose-300 dark:bg-rose-900/30 dark:text-rose-300 dark:ring-rose-700'
              : 'bg-gray-100 text-slate-500 hover:bg-gray-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'"
            @click="groupBy = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <EmptyState
        v-if="!summary || summary.total_seconds === 0"
        title="No activity tracked"
        description="Your browsing time will appear here once the extension starts tracking."
        icon="chart"
        color="rose"
      />

      <template v-else>
        <!-- Domain breakdown -->
        <div class="mb-6 rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-colors dark:border-slate-700 dark:bg-slate-800/90">
          <h3 class="mb-3 text-xs font-black uppercase tracking-wider text-slate-500">Top Domains</h3>
          <ul class="space-y-2">
            <li
              v-for="(domain, index) in summary.domains"
              :key="domain.domain"
              class="flex items-center gap-3"
            >
              <img
                :src="`https://www.google.com/s2/favicons?domain=${domain.domain}&sz=16`"
                alt=""
                class="h-4 w-4 shrink-0 rounded-sm"
                @error="($event.target as HTMLImageElement).style.visibility = 'hidden'"
              />
              <span class="w-28 shrink-0 truncate text-sm font-medium text-slate-700 dark:text-slate-300" :title="domain.domain">
                {{ domain.domain }}
              </span>
              <div class="flex h-5 flex-1 overflow-hidden rounded-sm bg-gray-100 dark:bg-slate-700">
                <div
                  class="h-full rounded-sm transition-all"
                  :class="barColor(index)"
                  :style="{ width: `${Math.max(domain.percentage, 2)}%` }"
                />
              </div>
              <span class="w-16 text-right text-xs font-semibold tabular-nums text-slate-600 dark:text-slate-400">
                {{ formatDuration(domain.total_seconds) }}
              </span>
              <span class="w-12 text-right text-[11px] tabular-nums text-slate-400">
                {{ domain.percentage }}%
              </span>
            </li>
          </ul>
        </div>

        <!-- Trend chart -->
        <div v-if="summary.timeline.length > 0" class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-colors dark:border-slate-700 dark:bg-slate-800/90">
          <h3 class="mb-3 text-xs font-black uppercase tracking-wider text-slate-500">
            Trend · {{ groupBy }}
          </h3>
          <div class="flex items-end justify-between gap-1" :style="{ height: `${chartMaxHeight + 28}px` }">
            <div
              v-for="(point, index) in summary.timeline"
              :key="point.period"
              class="flex flex-1 flex-col items-center justify-end"
              :title="formatDuration(point.total_seconds)"
            >
              <span class="mb-1 text-[10px] font-medium tabular-nums text-slate-500">
                {{ formatDuration(point.total_seconds) }}
              </span>
              <div
                class="w-full rounded-t-sm bg-rose-500/60 transition hover:bg-rose-400/80"
                :style="{ height: maxTimelineValue > 0 ? `${(point.total_seconds / maxTimelineValue) * chartMaxHeight}px` : '0px' }"
              />
              <span
                class="mt-1 text-[9px] leading-tight text-slate-400"
                :class="{ 'opacity-0': point.total_seconds === 0 }"
              >
                {{ timelineLabels[index] }}
              </span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useUser } from '../composables/useUser'
import { useAnalytics } from '../composables/useAnalytics'
import { formatDuration } from '../lib/utils'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import UserSelector from './ui/UserSelector.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import EmptyState from './ui/EmptyState.vue'
import SelectUserPrompt from './ui/SelectUserPrompt.vue'
import type { DatePreset, GroupBy } from '../composables/useAnalytics'

const { users, currentUserId, setUser, clearUser } = useUser()

const selectedUserId = ref<number | null>(currentUserId.value)

const {
  summary,
  isLoading,
  error,
  datePreset,
  customStart,
  customEnd,
  groupBy,
  totalDuration,
  domainCount,
  maxTimelineValue,
  timelineLabels,
  load,
} = useAnalytics(selectedUserId)

const chartMaxHeight = 160

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

const barColors = [
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

function barColor(index: number): string {
  return barColors[index % barColors.length]
}

watch(selectedUserId, (id) => {
  if (id) {
    setUser(id)
    load()
  } else {
    clearUser()
  }
})

if (selectedUserId.value) {
  setUser(selectedUserId.value)
  load()
}
</script>
