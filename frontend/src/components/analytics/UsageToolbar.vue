<template>
  <div
    class="mb-4 rounded-2xl border border-gray-200/80 bg-white/90 p-3 shadow-sm transition-colors dark:border-slate-700/80 dark:bg-slate-800/90"
  >
    <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <!-- Date presets -->
      <div
        class="flex scrollbar-none items-center gap-1 overflow-x-auto rounded-xl bg-slate-100/80 p-1 dark:bg-slate-700/50"
        role="group"
        aria-label="Date range"
      >
        <button
          v-for="preset in DATE_PRESETS"
          :key="preset.value"
          type="button"
          class="shrink-0 rounded-lg px-3 py-1.5 text-xs font-bold transition"
          :class="
            datePreset === preset.value
              ? 'bg-white text-rose-700 shadow-sm dark:bg-slate-900 dark:text-rose-300'
              : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
          "
          :aria-pressed="datePreset === preset.value"
          @click="$emit('update:datePreset', preset.value)"
        >
          {{ preset.label }}
        </button>

        <template v-if="datePreset === 'custom'">
          <span class="mx-1 hidden h-5 w-px bg-slate-300/80 lg:block dark:bg-slate-600" />
          <input
            :value="customStart"
            type="date"
            aria-label="Range start"
            class="w-32 rounded-lg border border-gray-200 bg-white px-2.5 py-1.5 text-xs font-semibold text-slate-700 transition outline-none focus:border-rose-400 focus:ring-2 focus:ring-rose-100 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:focus:ring-rose-900/30"
            @input="$emit('update:customStart', ($event.target as HTMLInputElement).value)"
          />
          <ArrowRight class="h-3.5 w-3.5 shrink-0 text-slate-400" aria-hidden="true" />
          <input
            :value="customEnd"
            type="date"
            aria-label="Range end"
            class="w-32 rounded-lg border border-gray-200 bg-white px-2.5 py-1.5 text-xs font-semibold text-slate-700 transition outline-none focus:border-rose-400 focus:ring-2 focus:ring-rose-100 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:focus:ring-rose-900/30"
            @input="$emit('update:customEnd', ($event.target as HTMLInputElement).value)"
          />
        </template>
      </div>

      <!-- Grouping (hidden for single-day range) -->
      <div v-if="datePreset !== 'today'" class="flex items-center gap-2">
        <span class="text-[11px] font-medium text-slate-500 dark:text-slate-400">Group by</span>
        <div
          class="flex items-center gap-0.5 rounded-lg bg-slate-100 p-0.5 dark:bg-slate-800"
          role="group"
          aria-label="Group by"
        >
          <button
            v-for="opt in GROUP_OPTIONS"
            :key="opt.value"
            type="button"
            class="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-[11px] font-bold transition sm:py-1"
            :class="
              groupBy === opt.value
                ? 'bg-white text-rose-700 shadow-sm dark:bg-slate-700 dark:text-rose-300'
                : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
            "
            :aria-pressed="groupBy === opt.value"
            @click="$emit('update:groupBy', opt.value)"
          >
            <component
              :is="groupIcon(opt.value)"
              class="h-3.5 w-3.5"
              :stroke-width="2.25"
              aria-hidden="true"
            />
            {{ opt.label }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowRight, Calendar1, CalendarDays, CalendarRange, type LucideIcon } from '@lucide/vue';
import { DATE_PRESETS, GROUP_OPTIONS, type DatePreset, type GroupBy } from './analyticsFormat';

defineProps<{
  datePreset: DatePreset;
  customStart: string;
  customEnd: string;
  groupBy: GroupBy;
}>();

defineEmits<{
  'update:datePreset': [value: DatePreset];
  'update:customStart': [value: string];
  'update:customEnd': [value: string];
  'update:groupBy': [value: GroupBy];
}>();

const groupIcon = (g: GroupBy): LucideIcon =>
  g === 'day' ? Calendar1 : g === 'week' ? CalendarRange : CalendarDays;
</script>

<style scoped>
.scrollbar-none {
  scrollbar-width: none;
}
.scrollbar-none::-webkit-scrollbar {
  display: none;
}
</style>
