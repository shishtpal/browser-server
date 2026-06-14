import { ref, computed, watch, type Ref } from 'vue'
import { getAnalyticsSummary } from '../lib/api'
import { formatDuration } from '../lib/utils'
import type { AnalyticsSummary, DomainUsage, TimelinePoint } from '../types'

export type DatePreset = 'today' | '7days' | '30days' | 'custom'
export type GroupBy = 'day' | 'week' | 'month'

export function useAnalytics(selectedUserId: Ref<number | null>) {
  const summary = ref<AnalyticsSummary | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const datePreset = ref<DatePreset>('7days')
  const customStart = ref('')
  const customEnd = ref('')
  const groupBy = ref<GroupBy>('day')

  const todayStr = computed(() => new Date().toISOString().slice(0, 10))

  const dateRange = computed(() => {
    const today = todayStr.value
    switch (datePreset.value) {
      case 'today':
        return { start: today, end: today }
      case '7days': {
        const d = new Date()
        d.setDate(d.getDate() - 6)
        return { start: d.toISOString().slice(0, 10), end: today }
      }
      case '30days': {
        const d = new Date()
        d.setDate(d.getDate() - 29)
        return { start: d.toISOString().slice(0, 10), end: today }
      }
      case 'custom':
        return { start: customStart.value || today, end: customEnd.value || today }
    }
  })

  const totalDuration = computed(() => {
    if (!summary.value) return '0s'
    return formatDuration(summary.value.total_seconds)
  })

  const domainCount = computed(() => summary.value?.domains.length ?? 0)

  const maxTimelineValue = computed(() => {
    if (!summary.value?.timeline.length) return 0
    return Math.max(...summary.value.timeline.map((t) => t.total_seconds))
  })

  function periodLabel(period: string): string {
    if (groupBy.value === 'month') return period
    if (groupBy.value === 'week') {
      const [year, week] = period.split('-')
      return `W${week}`
    }
    const d = new Date(period)
    if (Number.isNaN(d.getTime())) return period
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  }

  const timelineLabels = computed(() => {
    return summary.value?.timeline.map((tp) => periodLabel(tp.period)) ?? []
  })

  const load = async () => {
    if (!selectedUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      const { start, end } = dateRange.value
      summary.value = await getAnalyticsSummary({
        user_id: selectedUserId.value,
        start_date: start,
        end_date: end,
        group_by: groupBy.value,
        limit: 10,
      })
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load usage data'
      summary.value = null
    } finally {
      isLoading.value = false
    }
  }

  watch(selectedUserId, (id) => {
    if (id) load()
  })

  watch([datePreset, groupBy, customStart, customEnd], () => {
    if (selectedUserId.value) load()
  })

  if (selectedUserId.value) load()

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
    load,
  }
}
