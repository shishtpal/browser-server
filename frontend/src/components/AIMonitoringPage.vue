<template>
  <main class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
    <PageHeader badge="Operations" title="AI Monitor" color="cyan">
      <template #actions>
        <div class="flex flex-wrap items-center gap-2">
          <label class="text-xs font-bold text-slate-600 dark:text-slate-300"
            >Metrics window
            <select
              v-model.number="windowHours"
              class="ml-1 rounded-lg border border-slate-300 bg-white px-2 py-1.5 dark:border-white/10 dark:bg-slate-800"
              @change="refresh"
            >
              <option :value="24">24 hours</option>
              <option :value="168">7 days</option>
              <option :value="720">30 days</option>
              <option :value="2160">90 days</option>
            </select>
          </label>
          <Button variant="secondary" size="sm" :loading="isLoading" @click="refresh"
            >Refresh</Button
          >
          <span class="text-xs text-slate-500"
            >Last activity: {{ formatDate(metrics?.latest_activity) }}</span
          >
        </div>
      </template>
    </PageHeader>

    <ErrorBanner v-if="error" :message="error" :on-retry="refresh" />
    <LoadingSpinner
      v-if="isLoading && !metrics && !logs.length"
      message="Loading AI monitoring…"
      color="cyan"
    />
    <template v-else>
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-8">
        <StatCard :value="formatNumber(metrics?.requests)" label="Requests" />
        <StatCard
          :value="`${successRate.toFixed(1)}%`"
          label="Success"
          variant="primary"
          color="emerald"
        />
        <StatCard
          :value="formatNumber(metrics?.errors)"
          :label="`${errorRate}% errors`"
          variant="primary"
          color="rose"
        />
        <StatCard
          :value="formatNumber(metrics?.cancellations)"
          label="Cancelled"
          variant="primary"
          color="amber"
        />
        <StatCard
          :value="formatNumber(metrics?.total_tokens)"
          label="Total tokens"
          variant="primary"
          color="violet"
        />
        <StatCard
          :value="formatDuration(metrics?.average_latency_ms || 0)"
          label="Avg latency"
          variant="primary"
          color="cyan"
        />
        <StatCard
          :value="formatDuration(metrics?.max_latency_ms || 0)"
          label="Max latency"
          variant="primary"
          color="indigo"
        />
        <StatCard
          :value="formatNumber(toolActivity)"
          label="Tool activity"
          variant="primary"
          color="amber"
        />
      </div>

      <form
        class="mt-5 grid gap-3 rounded-xl border border-slate-200 bg-white p-4 sm:grid-cols-2 lg:grid-cols-5 dark:border-white/10 dark:bg-slate-900"
        @submit.prevent="applyIdFilters"
      >
        <label class="text-xs font-bold"
          >Source<select v-model="source" class="filter-control" @change="refresh">
            <option value="">All sources</option>
            <option value="chat">Chat</option>
            <option value="task_agent">Task agent</option>
          </select></label
        >
        <label class="text-xs font-bold"
          >Status<select v-model="status" class="filter-control" @change="refresh">
            <option value="">All statuses</option>
            <option value="success">Success</option>
            <option value="error">Error</option>
            <option value="cancelled">Cancelled</option>
          </select></label
        >
        <label class="text-xs font-bold"
          >Conversation ID<input
            v-model="conversationInput"
            class="filter-control"
            placeholder="Optional ID"
        /></label>
        <label class="text-xs font-bold"
          >Task ID<input v-model="taskInput" class="filter-control" placeholder="Optional ID"
        /></label>
        <div class="flex items-end gap-2">
          <Button type="submit" size="sm">Apply</Button
          ><Button type="button" variant="ghost" size="sm" @click="clearFilters">Clear</Button>
        </div>
      </form>

      <EmptyState
        v-if="!logs.length"
        class="mt-5"
        title="No AI requests found"
        description="Try another window or clear the active filters."
        icon="search"
        color="cyan"
      />
      <div v-else class="mt-5">
        <div
          class="hidden overflow-x-auto rounded-xl border border-slate-200 bg-white md:block dark:border-white/10 dark:bg-slate-900"
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
                class="cursor-pointer hover:bg-cyan-50 focus:bg-cyan-50 focus:ring-2 focus:ring-cyan-500 focus:outline-none focus:ring-inset dark:hover:bg-cyan-950/20 dark:focus:bg-cyan-950/20"
                @click="selected = log"
                @keydown.enter="selected = log"
                @keydown.space.prevent="selected = log"
              >
                <td class="px-3 py-3 whitespace-nowrap">{{ formatDate(log.created_at) }}</td>
                <td class="px-3 py-3">{{ sourceLabel(log.source) }}</td>
                <td class="max-w-52 truncate px-3 py-3" :title="`${log.provider} / ${log.model}`">
                  {{ log.provider }} / {{ log.model }}
                </td>
                <td class="px-3 py-3">
                  <span :class="statusClass(log.status)" class="rounded-full px-2 py-1 font-bold">{{
                    log.status
                  }}</span>
                </td>
                <td class="px-3 py-3">{{ formatDuration(log.latency_ms) }}</td>
                <td class="px-3 py-3">{{ formatOptionalNumber(log.total_tokens) }}</td>
                <td class="px-3 py-3">{{ log.tool_calls?.length || 0 }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="space-y-3 md:hidden">
          <button
            v-for="log in logs"
            :key="log.id"
            type="button"
            class="w-full rounded-xl border border-slate-200 bg-white p-4 text-left focus:ring-2 focus:ring-cyan-500 focus:outline-none dark:border-white/10 dark:bg-slate-900"
            @click="selected = log"
          >
            <div class="flex justify-between gap-3">
              <strong>{{ log.provider }} / {{ log.model }}</strong
              ><span
                :class="statusClass(log.status)"
                class="rounded-full px-2 py-1 text-xs font-bold"
                >{{ log.status }}</span
              >
            </div>
            <div class="mt-2 grid grid-cols-2 gap-2 text-xs text-slate-500">
              <span>{{ formatDate(log.created_at) }}</span
              ><span>{{ sourceLabel(log.source) }}</span
              ><span>{{ formatDuration(log.latency_ms) }}</span
              ><span
                >{{ formatOptionalNumber(log.total_tokens) }} tokens ·
                {{ log.tool_calls?.length || 0 }} tools</span
              >
            </div>
          </button>
        </div>
        <div v-if="hasMore" class="mt-4 text-center">
          <Button :loading="isLoadingMore" variant="secondary" @click="loadMore">Load more</Button>
        </div>
      </div>
    </template>
    <AIRequestDetail
      :request="selected"
      :can-go-prev="canGoPrev"
      :can-go-next="canGoNext"
      @close="selected = null"
      @prev="moveSelected(-1)"
      @next="moveSelected(1)"
    />
  </main>
</template>

<script setup lang="ts">
import type { AIRequestLog } from '@browser-server/shared-types'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAIMonitoring } from '../composables/useAIMonitoring'
import AIRequestDetail from './ai-monitoring/AIRequestDetail.vue'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import Button from './ui/Button.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import EmptyState from './ui/EmptyState.vue'

const monitor = useAIMonitoring()

const {
  metrics,
  logs,
  windowHours,
  source,
  status,
  conversationInput,
  taskInput,
  isLoading,
  isLoadingMore,
  error,
  hasMore,
  successRate,
  refresh,
  loadMore,
  applyIdFilters,
  clearFilters,
} = monitor

const selected = ref<AIRequestLog | null>(null)

const headings = ['Timestamp', 'Source', 'Provider / model', 'Status', 'Latency', 'Tokens', 'Tools']

const selectedIndex = computed(() => {
  if (!selected.value) return -1
  return logs.value.findIndex((log) => log.id === selected.value?.id)
})

const canGoPrev = computed(() => selectedIndex.value > 0)

const canGoNext = computed(
  () => selectedIndex.value >= 0 && selectedIndex.value < logs.value.length - 1,
)

const moveSelected = (direction: -1 | 1) => {
  if (!selected.value) return
  const index = selectedIndex.value
  if (index < 0) return
  const nextIndex = index + direction
  if (nextIndex < 0 || nextIndex >= logs.value.length) return
  selected.value = logs.value[nextIndex]
}

const toolActivity = computed(
  () =>
    (metrics.value?.tool_successes || 0) +
    (metrics.value?.tool_errors || 0) +
    (metrics.value?.tool_rejections || 0),
)
const errorRate = computed(() =>
  metrics.value?.requests
    ? ((metrics.value.errors / metrics.value.requests) * 100).toFixed(1)
    : '0.0',
)

const formatNumber = (value?: number) => (value || 0).toLocaleString()
const formatOptionalNumber = (value?: number) =>
  value === undefined ? '—' : value.toLocaleString()
const formatDate = (value?: string) => (value ? new Date(value).toLocaleString() : 'No activity')
const formatDuration = (ms: number) =>
  ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${Math.round(ms)}ms`
const sourceLabel = (value: string) => (value === 'task_agent' ? 'Task agent' : 'Chat')
const statusClass = (value: string) =>
  value === 'success'
    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
    : value === 'cancelled'
      ? 'bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
      : 'bg-rose-100 text-rose-700 dark:bg-rose-950 dark:text-rose-300'
const tokenChanged = () => refresh()

onMounted(() => {
  refresh()
  window.addEventListener('api-token-changed', tokenChanged)
})

onBeforeUnmount(() => window.removeEventListener('api-token-changed', tokenChanged))
</script>

<style scoped>
.filter-control {
  display: block;
  width: 100%;
  margin-top: 0.25rem;
  padding: 0.5rem 0.75rem;
  border: 1px solid rgb(203 213 225);
  border-radius: 0.5rem;
  background: white;
  color: rgb(15 23 42);
  font-size: 0.875rem;
  font-weight: 400;
}
.filter-control:focus {
  border-color: rgb(6 182 212);
  outline: 2px solid rgb(6 182 212 / 0.3);
}
:global(.dark) .filter-control {
  border-color: rgb(255 255 255 / 0.1);
  background: rgb(30 41 59);
  color: white;
}
</style>
