import type { AnalyticsSummary } from '../../../types';
import { computed, ref, watch, type Ref } from 'vue';
import { getAnalyticsSummary } from '../../../lib/api';
import { formatDuration } from '../../../lib/utils';
import { periodLabel, presetRange, type DatePreset, type GroupBy } from '../analyticsFormat';

export type { DatePreset, GroupBy } from '../analyticsFormat';

/**
 * Usage (domain-time) analytics for the selected user: summary fetch,
 * preset/custom date range, and grouping.
 *
 * Loading starts automatically (immediate watcher) whenever user, preset,
 * or grouping changes — no manual page bootstrapping.
 */
export function useAnalytics(selectedUserId: Ref<number | null>) {
  const summary = ref<AnalyticsSummary | null>(null);
  const isLoading = ref(false);
  const error = ref<string | null>(null);
  const datePreset = ref<DatePreset>('7days');
  const customStart = ref('');
  const customEnd = ref('');
  const groupBy = ref<GroupBy>('day');

  const dateRange = computed(() =>
    presetRange(datePreset.value, customStart.value, customEnd.value),
  );

  const totalDuration = computed(() =>
    summary.value ? formatDuration(summary.value.total_seconds) : '0s',
  );

  const domainCount = computed(() => summary.value?.domains.length ?? 0);

  const maxTimelineValue = computed(() => {
    if (!summary.value?.timeline.length) return 0;
    return Math.max(...summary.value.timeline.map((t) => t.total_seconds));
  });

  const timelineLabels = computed(
    () => summary.value?.timeline.map((tp) => periodLabel(tp.period, groupBy.value)) ?? [],
  );

  const isEmpty = computed(() => !summary.value || summary.value.total_seconds === 0);

  const load = async () => {
    if (!selectedUserId.value) return;
    isLoading.value = true;
    error.value = null;
    try {
      const { start, end } = dateRange.value;
      summary.value = await getAnalyticsSummary({
        user_id: selectedUserId.value,
        start_date: start,
        end_date: end,
        group_by: groupBy.value,
        limit: 10,
      });
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load usage data';
      summary.value = null;
    } finally {
      isLoading.value = false;
    }
  };

  watch(
    selectedUserId,
    (id) => {
      if (id) load();
      else summary.value = null;
    },
    { immediate: true },
  );

  watch([datePreset, groupBy, customStart, customEnd], () => {
    if (selectedUserId.value) load();
  });

  return {
    summary,
    isLoading,
    error,
    datePreset,
    customStart,
    customEnd,
    groupBy,
    dateRange,
    totalDuration,
    domainCount,
    maxTimelineValue,
    timelineLabels,
    isEmpty,
    load,
  };
}
