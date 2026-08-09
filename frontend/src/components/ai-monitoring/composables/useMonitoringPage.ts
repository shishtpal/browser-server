import type { AIRequestLog } from '@browser-server/shared-types';
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useAIMonitoring } from './useAIMonitoring';

/**
 * Orchestrates the AI Monitor page: selection + prev/next navigation on the
 * request list, and re-loading when the API token changes.
 */
export function useMonitoringPage() {
  const monitor = useAIMonitoring();
  const { logs } = monitor;

  const selected = ref<AIRequestLog | null>(null);

  const selectedIndex = computed(() => {
    if (!selected.value) return -1;
    return logs.value.findIndex((log) => log.id === selected.value?.id);
  });

  const canGoPrev = computed(() => selectedIndex.value > 0);
  const canGoNext = computed(
    () => selectedIndex.value >= 0 && selectedIndex.value < logs.value.length - 1,
  );

  const openDetail = (log: AIRequestLog) => {
    selected.value = log;
  };

  const closeDetail = () => {
    selected.value = null;
  };

  const moveSelected = (direction: -1 | 1) => {
    const index = selectedIndex.value;
    if (index < 0) return;
    const nextIndex = index + direction;
    if (nextIndex < 0 || nextIndex >= logs.value.length) return;
    selected.value = logs.value[nextIndex];
  };

  /** Reload both windows when the API token changes. */
  const onTokenChanged = () => monitor.refresh();
  onMounted(() => {
    monitor.refresh();
    window.addEventListener('api-token-changed', onTokenChanged);
  });
  onBeforeUnmount(() => window.removeEventListener('api-token-changed', onTokenChanged));

  return {
    monitor,
    selected,
    canGoPrev,
    canGoNext,
    openDetail,
    closeDetail,
    moveSelected,
  };
}
