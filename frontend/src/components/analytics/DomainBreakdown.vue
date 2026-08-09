<template>
  <section
    class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-colors dark:border-slate-700 dark:bg-slate-800/90"
    aria-labelledby="domain-breakdown-heading"
  >
    <div class="mb-3 flex items-center justify-between gap-2">
      <h3
        id="domain-breakdown-heading"
        class="flex items-center gap-1.5 text-xs font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
      >
        <Globe class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
        Top Domains
      </h3>
      <span class="text-[10px] font-semibold text-slate-400 tabular-nums dark:text-slate-500">
        {{ domains.length }} shown
      </span>
    </div>

    <ul class="space-y-2.5">
      <li
        v-for="(domain, index) in domains"
        :key="domain.domain"
        class="flex items-center gap-2.5 sm:gap-3"
      >
        <span
          class="w-4 shrink-0 text-right text-[10px] font-black text-slate-300 tabular-nums dark:text-slate-600"
        >
          {{ index + 1 }}
        </span>
        <img
          :src="domainFaviconUrl(domain.domain)"
          alt=""
          loading="lazy"
          class="h-4 w-4 shrink-0 rounded-sm bg-slate-100 dark:bg-slate-700"
          @error="hideBrokenIcon"
        />
        <span
          class="w-24 shrink-0 truncate text-sm font-medium text-slate-700 sm:w-32 dark:text-slate-300"
          :title="domain.domain"
        >
          {{ domain.domain }}
        </span>

        <!-- Bar -->
        <div
          class="flex h-5 min-w-0 flex-1 overflow-hidden rounded-md bg-gray-100 dark:bg-slate-700"
          role="img"
          :aria-label="`${domain.domain}: ${formatDuration(domain.total_seconds)} (${domain.percentage}%)`"
        >
          <div
            class="h-full rounded-md transition-all duration-500"
            :class="domainBarColor(index)"
            :style="{ width: `${Math.max(domain.percentage, 2)}%` }"
          />
        </div>

        <span
          class="w-14 shrink-0 text-right text-xs font-semibold text-slate-600 tabular-nums sm:w-16 dark:text-slate-400"
        >
          {{ formatDuration(domain.total_seconds) }}
        </span>
        <span
          class="hidden w-12 shrink-0 text-right text-[11px] text-slate-400 tabular-nums sm:block dark:text-slate-500"
        >
          {{ domain.percentage }}%
        </span>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import type { DomainUsage } from '../../types';
import { Globe } from '@lucide/vue';
import { formatDuration } from '../../lib/utils';
import { domainBarColor, domainFaviconUrl } from './analyticsFormat';

defineProps<{ domains: DomainUsage[] }>();

/** Broken favicons collapse to a neutral block instead of the alt icon. */
function hideBrokenIcon(event: Event) {
  (event.target as HTMLImageElement).style.visibility = 'hidden';
}
</script>
