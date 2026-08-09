<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Operations" title="AI Monitor" color="cyan">
      <template #stats>
        <StatCard :value="formatNumber(metrics?.requests)" label="Requests" color="cyan" />
        <StatCard
          :value="`${monitor.successRate.value.toFixed(1)}%`"
          label="Success"
          variant="primary"
          color="emerald"
        />
        <StatCard :value="`${errorRate}%`" label="Error rate" variant="primary" color="rose" />
        <StatCard
          :value="formatNumber(metrics?.total_tokens)"
          label="Tokens"
          variant="primary"
          color="violet"
        />
      </template>
      <template #controls>
        <div class="flex flex-wrap items-center gap-2">
          <label
            class="flex items-center gap-1.5 text-xs font-bold text-slate-600 dark:text-slate-300"
          >
            <Clock class="h-3.5 w-3.5 text-slate-400" :stroke-width="2.25" aria-hidden="true" />
            <span class="sr-only sm:not-sr-only">Window</span>
            <select
              v-model.number="windowHours"
              class="rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-xs font-semibold transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-100 focus:outline-none dark:border-white/10 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30"
              @change="refresh"
            >
              <option v-for="opt in WINDOW_OPTIONS" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </label>
          <Button variant="secondary" size="sm" :loading="isLoading" @click="refresh">
            <span class="inline-flex items-center gap-1.5">
              <RefreshCw class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
              Refresh
            </span>
          </Button>
          <span class="hidden text-xs text-slate-500 lg:inline dark:text-slate-400">
            Last activity: {{ lastActivity }}
          </span>
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
      <!-- Latency + activity row -->
      <div class="mb-5 grid grid-cols-2 gap-2 sm:grid-cols-4">
        <div
          v-for="card in activityCards"
          :key="card.label"
          class="flex items-center gap-2.5 rounded-xl border border-slate-200 bg-white p-3 shadow-sm dark:border-white/10 dark:bg-slate-900"
        >
          <span class="grid h-8 w-8 shrink-0 place-items-center rounded-lg" :class="card.bgClass">
            <component :is="card.icon" class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
          </span>
          <div class="min-w-0">
            <p class="truncate text-sm font-black text-slate-900 tabular-nums dark:text-white">
              {{ card.value }}
            </p>
            <p
              class="truncate text-[10px] font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
            >
              {{ card.label }}
            </p>
          </div>
        </div>
      </div>

      <RequestLogFilters
        v-model:source="source"
        v-model:status="status"
        v-model:conversation-input="conversationInput"
        v-model:task-input="taskInput"
        @apply="applyIdFilters"
        @clear="clearFilters"
      />

      <EmptyState
        v-if="!logs.length"
        class="mt-5"
        title="No AI requests found"
        description="Try another window or clear the active filters."
        icon="search"
        color="cyan"
      />

      <div v-else class="mt-5">
        <RequestLogList :logs="logs" :selected-id="selected?.id" @select="openDetail" />
        <div v-if="hasMore" class="mt-4 text-center">
          <Button
            variant="secondary"
            size="sm"
            :loading="isLoadingMore"
            loading-text="Loading…"
            @click="loadMore"
          >
            Load more
          </Button>
        </div>
      </div>
    </template>

    <AIRequestDetail
      :request="selected"
      :can-go-prev="canGoPrev"
      :can-go-next="canGoNext"
      @close="closeDetail"
      @prev="moveSelected(-1)"
      @next="moveSelected(1)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Ban, Clock, Gauge, RefreshCw, Timer, Wrench, type LucideIcon } from '@lucide/vue';
import { useMonitoringPage } from './ai-monitoring/composables/useMonitoringPage';
import {
  formatMs,
  formatNumber,
  formatOptionalNumber,
  formatTimestamp,
  WINDOW_OPTIONS,
} from './ai-monitoring/monitoringFormat';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import Button from './ui/Button.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import EmptyState from './ui/EmptyState.vue';
import RequestLogFilters from './ai-monitoring/RequestLogFilters.vue';
import RequestLogList from './ai-monitoring/RequestLogList.vue';
import AIRequestDetail from './ai-monitoring/AIRequestDetail.vue';

const { monitor, selected, canGoPrev, canGoNext, openDetail, closeDetail, moveSelected } =
  useMonitoringPage();

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
  refresh,
  loadMore,
  applyIdFilters,
  clearFilters,
} = monitor;

const errorRate = computed(() =>
  metrics.value?.requests
    ? ((metrics.value.errors / metrics.value.requests) * 100).toFixed(1)
    : '0.0',
);

const toolActivity = computed(
  () =>
    (metrics.value?.tool_errors || 0) +
    (metrics.value?.tool_rejections || 0) +
    (metrics.value?.tool_successes || 0),
);

const lastActivity = computed(() => formatTimestamp(metrics.value?.latest_activity));

const activityCards = computed<
  Array<{ label: string; value: string; icon: LucideIcon; bgClass: string }>
>(() => [
  {
    label: 'Cancelled',
    value: formatNumber(metrics.value?.cancellations),
    icon: Ban,
    bgClass: 'bg-amber-50 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400',
  },
  {
    label: 'Avg latency',
    value: formatMs(metrics.value?.average_latency_ms),
    icon: Timer,
    bgClass: 'bg-cyan-50 text-cyan-600 dark:bg-cyan-900/20 dark:text-cyan-400',
  },
  {
    label: 'Max latency',
    value: formatMs(metrics.value?.max_latency_ms),
    icon: Gauge,
    bgClass: 'bg-violet-50 text-violet-600 dark:bg-violet-900/20 dark:text-violet-400',
  },
  {
    label: 'Tool activity',
    value: formatOptionalNumber(toolActivity.value),
    icon: Wrench,
    bgClass: 'bg-rose-50 text-rose-600 dark:bg-rose-900/20 dark:text-rose-400',
  },
]);
</script>
