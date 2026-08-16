<template>
  <form
    class="self-start rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-colors lg:sticky lg:top-4 dark:border-white/10 dark:bg-slate-800/90"
    @submit.prevent="$emit('submit')"
  >
    <div class="mb-1 flex items-baseline justify-between">
      <label
        for="video-prompt"
        class="text-xs font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
      >
        Prompt
      </label>
      <span class="text-[10px] font-semibold text-slate-400 tabular-nums dark:text-slate-500">
        {{ prompt.length }}
      </span>
    </div>
    <textarea
      id="video-prompt"
      ref="textareaRef"
      :value="prompt"
      rows="6"
      placeholder="Describe the video you want to create — subject, action, scene, camera movement, lighting, style"
      class="w-full resize-y rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition placeholder:text-slate-400 focus:border-cyan-400 focus:ring-4 focus:ring-cyan-100 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:ring-cyan-900/30"
      @input="$emit('update:prompt', ($event.target as HTMLTextAreaElement).value)"
      @keydown.enter.meta.prevent="$emit('submit')"
      @keydown.enter.ctrl.prevent="$emit('submit')"
    />
    <p class="mt-1 text-[10px] font-semibold text-slate-400 dark:text-slate-500">
      <kbd class="rounded border border-gray-300 px-1 dark:border-slate-600">Ctrl</kbd> +
      <kbd class="rounded border border-gray-300 px-1 dark:border-slate-600">Enter</kbd> to generate
    </p>

    <div class="mt-3 space-y-3">
      <div v-if="providerNames.length > 1">
        <label for="video-provider" :class="labelClass">Provider</label>
        <select
          id="video-provider"
          :value="provider"
          :class="selectClass"
          @change="$emit('update:provider', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="name in providerNames" :key="name" :value="name">{{ name }}</option>
        </select>
      </div>

      <div>
        <label for="video-model" :class="labelClass">Model</label>
        <select
          id="video-model"
          :value="model"
          :class="selectClass"
          @change="$emit('update:model', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="m in models" :key="m.id" :value="m.id">{{ m.label }}</option>
        </select>
      </div>

      <div
        v-for="group in groups"
        :key="group.name"
        class="rounded-lg border border-gray-200 dark:border-slate-700"
      >
        <button
          type="button"
          class="flex w-full items-center justify-between px-3 py-2 text-xs font-black tracking-wider text-slate-500 uppercase dark:text-slate-400"
          @click="toggle(group.name)"
        >
          <span>{{ group.name }}</span>
          <ChevronDown
            class="h-4 w-4 transition-transform"
            :class="openGroups.has(group.name) ? '' : '-rotate-90'"
            :stroke-width="2.5"
            aria-hidden="true"
          />
        </button>
        <div v-if="openGroups.has(group.name)" class="space-y-3 px-3 pb-3">
          <ParamField
            v-for="spec in group.specs"
            :key="spec.key"
            :spec="spec"
            :model-value="params[spec.key]"
            @update:model-value="setParam(spec.key, $event)"
          />
        </div>
      </div>
    </div>

    <Button
      type="submit"
      variant="gradient-cyan"
      size="lg"
      class="mt-4 w-full"
      :loading="busy"
      loading-text="Generating…"
      :disabled="!prompt.trim()"
    >
      Generate video
    </Button>
  </form>
</template>

<script setup lang="ts">
import type { VideoParamSpec } from '@browser-server/shared-types';
import { ref, watch } from 'vue';
import { ChevronDown } from '@lucide/vue';
import Button from '../ui/Button.vue';
import ParamField from './ParamField.vue';

const props = defineProps<{
  prompt: string;
  provider: string;
  model: string;
  params: Record<string, unknown>;
  providerNames: string[];
  models: { id: string; label: string }[];
  parameters: VideoParamSpec[];
  busy: boolean;
}>();

const emit = defineEmits<{
  'update:prompt': [value: string];
  'update:provider': [value: string];
  'update:model': [value: string];
  'update:params': [value: Record<string, unknown>];
  submit: [];
}>();

const textareaRef = ref<HTMLTextAreaElement | null>(null);
defineExpose({ focus: () => textareaRef.value?.focus() });

const labelClass =
  'mb-1 block text-xs font-black uppercase tracking-wider text-slate-500 dark:text-slate-400';
const selectClass =
  'w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition focus:border-cyan-400 focus:outline-none focus:ring-4 focus:ring-cyan-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:focus:ring-cyan-900/30';

interface Group {
  name: string;
  specs: VideoParamSpec[];
}

const groups = ref<Group[]>([]);
const openGroups = ref<Set<string>>(new Set());

// Recompute groups whenever the parameter list changes, preserving the first
// group open by default so the most common options are visible immediately.
function recomputeGroups(specs: VideoParamSpec[]) {
  const order: string[] = [];
  const map = new Map<string, VideoParamSpec[]>();
  for (const spec of specs) {
    if (spec.key === 'prompt') continue; // prompt has its own dedicated textarea above
    const name = spec.group || 'Options';
    if (!map.has(name)) {
      map.set(name, []);
      order.push(name);
    }
    map.get(name)!.push(spec);
  }
  groups.value = order.map((name) => ({ name, specs: map.get(name)! }));
  if (order.length && !openGroups.value.has(order[0])) {
    openGroups.value = new Set([order[0], ...openGroups.value]);
  }
}
recomputeGroups(props.parameters);

// Watch for model changes (new parameter set) from the parent.
watch(
  () => props.parameters,
  (specs) => recomputeGroups(specs),
);

function toggle(name: string) {
  const next = new Set(openGroups.value);
  if (next.has(name)) next.delete(name);
  else next.add(name);
  openGroups.value = next;
}

function setParam(key: string, value: unknown) {
  emit('update:params', { ...props.params, [key]: value });
}
</script>
