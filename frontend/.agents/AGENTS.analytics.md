# AGENTS.analytics.md — Usage / Analytics Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers `components/analytics/`.

`AnalyticsPage.vue` (the "Usage" page) wires `analytics/composables/useAnalytics.ts` (summary fetch; preset/custom range; day/week/month grouping; immediate load) and renders `UsageToolbar.vue` + `DomainBreakdown.vue` + `UsageTrendChart.vue` built from `analyticsFormat.ts` constants (presets, group icons, bar palette).

```
../components/analytics/
├── analyticsFormat.ts           # Date presets, group options, bar palette, period labels
├── composables/useAnalytics.ts  # Summary fetch + range/grouping (immediate load)
├── UsageToolbar.vue             # Presets + custom range + day/week/month segmented control
├── DomainBreakdown.vue          # Ranked top-domain bars (favicons, durations, %)
└── UsageTrendChart.vue          # Accessible bar chart (a11y label summarizes the series)
```
