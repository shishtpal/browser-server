<template>
  <div class="group relative">
    <component
      :is="icon"
      class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-slate-400 transition-colors group-focus-within:text-indigo-500"
      aria-hidden="true"
    />
    <select
      v-model="model"
      :aria-label="label"
      class="w-full cursor-pointer appearance-none rounded-xl border border-slate-200/80 bg-white/80 py-1.5 pr-7 pl-8 text-xs font-semibold text-slate-600 shadow-sm transition-all hover:border-indigo-300 hover:shadow focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none dark:border-slate-600/60 dark:bg-slate-700/60 dark:text-slate-300 dark:hover:border-indigo-500/50 dark:focus:border-indigo-500"
    >
      <option value="">{{ label }}</option>
      <option v-for="opt in options" :key="String(opt.value)" :value="opt.value">
        {{ opt.label }}
      </option>
    </select>
    <ChevronDown
      class="pointer-events-none absolute top-1/2 right-2 h-3 w-3 -translate-y-1/2 text-slate-400"
      aria-hidden="true"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ChevronDown, type LucideIcon } from '@lucide/vue';

defineProps<{
  label: string;
  icon: LucideIcon;
  options: { value: string; label: string }[];
}>();

/**
 * The bound value is `string | null`; the native select can't represent null,
 * so the empty-string option (shown as the label) stands in for it.
 */
const raw = defineModel<string | null>({ default: null });

const model = computed<string>({
  get: () => raw.value ?? '',
  set: (v: string) => (raw.value = v || null),
});
</script>
