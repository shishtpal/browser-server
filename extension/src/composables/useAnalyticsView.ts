import type { AnalyticsSummary, BrowserServerClient } from '@browser-server/shared-client'
import { computed, ref, type Ref } from 'vue'

export type DatePreset = 'today' | '7days' | '30days' | 'custom'
export type GroupBy = 'day' | 'week' | 'month'

export function useAnalyticsView(client: Ref<BrowserServerClient | null>, userId: Ref<number>) {
  const summary = ref<AnalyticsSummary | null>(null)
  const datePreset = ref<DatePreset>('today')
  const customStart = ref('')
  const customEnd = ref('')
  const groupBy = ref<GroupBy>('day')
  const isLoading = ref(false)
  const errorMessage = ref<string | null>(null)

  const todayStr = computed(() => new Date().toISOString().slice(0, 10))

  const dateRange = computed<{ start: string; end: string }>(() => {
    const today = todayStr.value
    const end = today

    switch (datePreset.value) {
      case 'today':
        return { start: today, end }
      case '7days': {
        const d = new Date()
        d.setDate(d.getDate() - 6)
        return { start: d.toISOString().slice(0, 10), end }
      }
      case '30days': {
        const d = new Date()
        d.setDate(d.getDate() - 29)
        return { start: d.toISOString().slice(0, 10), end }
      }
      case 'custom':
        return { start: customStart.value || today, end: customEnd.value || today }
    }
  })

  function formatPeriodLabel(period: string): string {
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
    return summary.value?.timeline.map((tp) => formatPeriodLabel(tp.period)) ?? []
  })

  async function load(): Promise<void> {
    if (!client.value || !userId.value) {
      return
    }

    isLoading.value = true
    errorMessage.value = null

    try {
      const { start, end } = dateRange.value
      summary.value = await client.value.getAnalyticsSummary({
        user_id: userId.value,
        start_date: start,
        end_date: end,
        group_by: groupBy.value,
        limit: 10,
      })
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error'
      errorMessage.value = `Server not reachable. ${message}`
      summary.value = null
    } finally {
      isLoading.value = false
    }
  }

  return {
    summary,
    datePreset,
    customStart,
    customEnd,
    groupBy,
    isLoading,
    errorMessage,
    todayStr,
    dateRange,
    timelineLabels,
    load,
  }
}
