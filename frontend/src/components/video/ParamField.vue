<template>
  <div class="space-y-1">
    <label :for="`vid-param-${spec.key}`" class="flex items-baseline justify-between gap-2">
      <span class="text-xs font-black tracking-wider text-slate-500 uppercase dark:text-slate-400">
        {{ spec.label }}
        <span v-if="spec.required" class="text-red-500">*</span>
      </span>
      <span v-if="spec.help" class="text-[10px] font-medium text-slate-400 dark:text-slate-500">
        {{ spec.help }}
      </span>
    </label>

    <!-- text -->
    <input
      v-if="spec.type === 'text'"
      :id="`vid-param-${spec.key}`"
      :value="stringValue"
      type="text"
      :placeholder="placeholder"
      class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30"
      @input="onText($event)"
    />

    <!-- textarea -->
    <textarea
      v-else-if="spec.type === 'textarea'"
      :id="`vid-param-${spec.key}`"
      :value="stringValue"
      rows="3"
      :placeholder="placeholder"
      class="w-full resize-y rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30"
      @input="onText($event)"
    />

    <!-- number -->
    <input
      v-else-if="spec.type === 'number'"
      :id="`vid-param-${spec.key}`"
      :value="numberValue"
      type="number"
      :min="spec.min"
      :max="spec.max"
      :step="spec.step ?? 1"
      class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30"
      @input="onNumber($event)"
    />

    <!-- select -->
    <select
      v-else-if="spec.type === 'select'"
      :id="`vid-param-${spec.key}`"
      :value="stringValue"
      class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30"
      @change="onSelect($event)"
    >
      <option v-for="opt in spec.options ?? []" :key="opt" :value="opt">{{ opt }}</option>
    </select>

    <!-- boolean -->
    <label v-else-if="spec.type === 'boolean'" class="flex items-center gap-2 py-1">
      <input
        :id="`vid-param-${spec.key}`"
        type="checkbox"
        :checked="boolValue"
        class="h-4 w-4 rounded border-gray-300 text-cyan-600 focus:ring-cyan-400"
        @change="onBool($event)"
      />
      <span class="text-xs font-semibold text-slate-600 dark:text-slate-300">Enabled</span>
    </label>

    <!-- image_urls -->
    <div v-else-if="spec.type === 'image_urls'" class="space-y-1.5">
      <div v-for="(url, i) in urlList" :key="i" class="flex items-center gap-1.5">
        <input
          :id="`vid-param-${spec.key}-${i}`"
          :value="url"
          type="url"
          placeholder="https://example.com/image.png"
          class="min-w-0 flex-1 rounded-lg border border-gray-300 bg-gray-50 px-2.5 py-1.5 text-xs font-semibold text-slate-700 shadow-sm transition focus:border-cyan-400 focus:ring-2 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30"
          @input="onUrlInput(i, $event)"
        />
        <button
          type="button"
          class="grid h-7 w-7 shrink-0 place-items-center rounded bg-red-100 text-red-600 transition hover:bg-red-200 dark:bg-red-900/40 dark:text-red-300"
          aria-label="Remove image URL"
          @click="removeUrl(i)"
        >
          <X class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        </button>
      </div>
      <button
        type="button"
        class="inline-flex items-center gap-1 rounded-md border border-dashed border-gray-300 px-2 py-1 text-[11px] font-bold text-slate-500 transition hover:border-cyan-400 hover:text-cyan-600 dark:border-slate-600 dark:text-slate-400"
        @click="addUrl"
      >
        <Plus class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
        Add image URL
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { VideoParamSpec } from '@browser-server/shared-types';
import { computed } from 'vue';
import { Plus, X } from '@lucide/vue';

const props = defineProps<{
  spec: VideoParamSpec;
  modelValue: unknown;
  placeholder?: string;
}>();

const emit = defineEmits<{ 'update:modelValue': [value: unknown] }>();

const stringValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''));
const numberValue = computed(() => {
  const n = Number(props.modelValue);
  return Number.isFinite(n) ? n : '';
});
const boolValue = computed(() => props.modelValue === true);

const urlList = computed<string[]>(() =>
  Array.isArray(props.modelValue) ? (props.modelValue as unknown[] as string[]) : [],
);

function onText(e: Event) {
  emit('update:modelValue', (e.target as HTMLInputElement | HTMLTextAreaElement).value);
}
function onNumber(e: Event) {
  const raw = (e.target as HTMLInputElement).value;
  emit('update:modelValue', raw === '' ? undefined : Number(raw));
}
function onSelect(e: Event) {
  emit('update:modelValue', (e.target as HTMLSelectElement).value);
}
function onBool(e: Event) {
  emit('update:modelValue', (e.target as HTMLInputElement).checked);
}

function addUrl() {
  emit('update:modelValue', [...urlList.value, '']);
}
function removeUrl(i: number) {
  const next = urlList.value.slice();
  next.splice(i, 1);
  emit('update:modelValue', next);
}
function onUrlInput(i: number, e: Event) {
  const next = urlList.value.slice();
  next[i] = (e.target as HTMLInputElement).value;
  emit('update:modelValue', next);
}
</script>
