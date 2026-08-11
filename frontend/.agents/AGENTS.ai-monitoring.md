# AGENTS.ai-monitoring.md — AI Monitor Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers the AI request monitoring page in `components/ai-monitoring/`.

`AIMonitoringPage.vue` (the "AI Monitor" page) is thin wiring over `ai-monitoring/composables/useMonitoringPage.ts`, which owns request selection + prev/next navigation and reloads when the API token changes. The underlying `useAIMonitoring.ts` fetches the summary metrics (requests, success rate, error rate, cancellations, tokens) plus a paged request-log list, with a configurable time window and source/status/conversation/task filters. A request-version guard discards stale responses when filters change mid-flight.

```
../components/ai-monitoring/
├── monitoringFormat.ts           # formatNumber, source/status labels, colors
├── composables/
│   ├── useMonitoringPage.ts      # Page orchestration: selection + prev/next, token-change reload
│   └── useAIMonitoring.ts        # Metrics + paged request logs, window/filters, request-version guard
├── RequestLogFilters.vue         # Source (chat/task_agent), status, conversation/task filters
├── RequestLogList.vue            # Paged log rows with infinite scroll
├── AIRequestDetail.vue           # Detail drawer for one request
└── PayloadBlock.vue              # Collapsible request/response payload rendering
```

Data comes from `lib/api/ai.ts` — `getAIMonitoring(windowHours)` and `getAIRequestLogs(filters)` — backed by the shared client's `/api/ai/monitoring` endpoints.
