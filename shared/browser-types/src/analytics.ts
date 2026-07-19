export interface UsageEntry {
  domain: string
  date: string
  seconds: number
}

export interface UsageBatchRequest {
  user_id: number
  entries: UsageEntry[]
}

export interface UsageBatchResponse {
  upserted: number
}

export interface DomainUsage {
  domain: string
  total_seconds: number
  percentage: number
}

export interface TimelinePoint {
  period: string
  total_seconds: number
}

export interface AnalyticsSummary {
  total_seconds: number
  domains: DomainUsage[]
  timeline: TimelinePoint[]
}

export interface AnalyticsSummaryParams {
  user_id: number
  start_date: string
  end_date: string
  group_by?: 'day' | 'week' | 'month'
  limit?: number
}
