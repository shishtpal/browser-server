import type { AIMonitoring, AIRequestLog } from '@browser-server/shared-types';
import { computed, ref } from 'vue';
import { getAIMonitoring, getAIRequestLogs } from '../../../lib/api';

const PAGE_SIZE = 25;

export function useAIMonitoring() {
  const metrics = ref<AIMonitoring | null>(null);
  const logs = ref<AIRequestLog[]>([]);
  const windowHours = ref(24);
  const source = ref<'' | 'chat' | 'task_agent'>('');
  const status = ref<'' | 'success' | 'error' | 'cancelled'>('');
  const conversationInput = ref('');
  const taskInput = ref('');
  const conversationId = ref('');
  const taskId = ref('');
  const isLoading = ref(false);
  const isLoadingMore = ref(false);
  const error = ref('');
  const hasMore = ref(false);
  let requestVersion = 0;

  const successRate = computed(() => {
    if (!metrics.value?.requests) return 0;
    const unsuccessful = metrics.value.errors + metrics.value.cancellations;
    return Math.max(0, ((metrics.value.requests - unsuccessful) / metrics.value.requests) * 100);
  });

  const logFilters = (offset: number) => ({
    source: source.value || undefined,
    status: status.value || undefined,
    conversationId: conversationId.value || undefined,
    taskId: taskId.value || undefined,
    limit: PAGE_SIZE,
    offset,
  });

  async function refresh() {
    const version = ++requestVersion;
    const hadData = logs.value.length > 0 || metrics.value !== null;
    isLoading.value = true;
    error.value = '';
    try {
      const [summary, page] = await Promise.all([
        getAIMonitoring(windowHours.value),
        getAIRequestLogs(logFilters(0)),
      ]);
      if (version !== requestVersion) return;
      metrics.value = summary;
      logs.value = page.logs;
      hasMore.value = page.logs.length === PAGE_SIZE;
    } catch (cause) {
      if (version !== requestVersion) return;
      error.value = cause instanceof Error ? cause.message : 'Unable to load AI monitoring data.';
      if (!hadData) {
        logs.value = [];
        metrics.value = null;
      }
    } finally {
      if (version === requestVersion) isLoading.value = false;
    }
  }

  async function loadMore() {
    if (isLoadingMore.value || !hasMore.value) return;
    const version = requestVersion;
    isLoadingMore.value = true;
    error.value = '';
    try {
      const page = await getAIRequestLogs(logFilters(logs.value.length));
      if (version !== requestVersion) return;
      logs.value.push(...page.logs);
      hasMore.value = page.logs.length === PAGE_SIZE;
    } catch (cause) {
      if (version !== requestVersion) return;
      error.value = cause instanceof Error ? cause.message : 'Unable to load more requests.';
    } finally {
      isLoadingMore.value = false;
    }
  }

  function applyIdFilters() {
    conversationId.value = conversationInput.value.trim();
    taskId.value = taskInput.value.trim();
    return refresh();
  }

  function clearFilters() {
    source.value = '';
    status.value = '';
    conversationInput.value = '';
    taskInput.value = '';
    conversationId.value = '';
    taskId.value = '';
    return refresh();
  }

  return {
    metrics,
    logs,
    windowHours,
    source,
    status,
    conversationInput,
    taskInput,
    conversationId,
    taskId,
    isLoading,
    isLoadingMore,
    error,
    hasMore,
    successRate,
    refresh,
    loadMore,
    applyIdFilters,
    clearFilters,
  };
}
