<template>
  <div class="space-y-5 p-4">
    <!-- Typography -->
    <section>
      <h3
        class="mb-2 text-[10px] font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
      >
        Typography
      </h3>
      <div class="space-y-3">
        <div>
          <label
            class="mb-1 flex items-center gap-1.5 text-[10px] font-semibold text-slate-600 dark:text-slate-400"
          >
            <Text class="h-3 w-3" :stroke-width="2.25" aria-hidden="true" />
            Font Family
          </label>
          <select
            :value="fontFamily"
            class="w-full rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs text-slate-700 dark:border-white/10 dark:bg-slate-900 dark:text-slate-200"
            @change="$emit('update:fontFamily', ($event.target as HTMLSelectElement).value)"
          >
            <option v-for="opt in FONT_OPTIONS" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>
        <div>
          <label class="mb-1 block text-[10px] font-semibold text-slate-600 dark:text-slate-400">
            Font Size
          </label>
          <div class="flex items-center gap-2">
            <input
              type="range"
              :value="fontSize"
              min="12"
              max="20"
              step="1"
              class="h-1.5 flex-1 cursor-pointer appearance-none rounded-full bg-slate-200 accent-indigo-600 dark:bg-slate-700"
              @input="$emit('update:fontSize', Number(($event.target as HTMLInputElement).value))"
            />
            <span
              class="w-8 text-center text-[10px] font-bold text-slate-600 tabular-nums dark:text-slate-400"
            >
              {{ fontSize }}px
            </span>
          </div>
        </div>
      </div>
    </section>

    <!-- Reasoning -->
    <section>
      <h3
        class="mb-2 text-[10px] font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
      >
        Thinking
      </h3>
      <label
        class="flex cursor-pointer items-start gap-2 text-xs text-slate-600 dark:text-slate-400"
      >
        <input
          :checked="showThinking"
          type="checkbox"
          class="mt-0.5 accent-indigo-600"
          @change="$emit('update:showThinking', ($event.target as HTMLInputElement).checked)"
        />
        <span>
          <span class="font-semibold text-slate-700 dark:text-slate-300">Show model thinking</span>
          <span class="mt-0.5 block text-[10px] text-slate-400 dark:text-slate-500">
            Display reasoning when the model provides it.
          </span>
        </span>
      </label>
    </section>
  </div>
</template>

<script setup lang="ts">
import { Text } from '@lucide/vue';

defineProps<{
  fontFamily: string;
  fontSize: number;
  showThinking: boolean;
}>();

defineEmits<{
  'update:fontFamily': [value: string];
  'update:fontSize': [value: number];
  'update:showThinking': [value: boolean];
}>();

const FONT_OPTIONS = [
  { value: 'system-ui', label: 'System Default' },
  { value: 'Inter, sans-serif', label: 'Inter' },
  { value: "'JetBrains Mono', monospace", label: 'JetBrains Mono' },
  { value: "'Fira Code', monospace", label: 'Fira Code' },
  { value: 'Georgia, serif', label: 'Georgia' },
  { value: 'Menlo, Monaco, monospace', label: 'Menlo / Monaco' },
];
</script>
