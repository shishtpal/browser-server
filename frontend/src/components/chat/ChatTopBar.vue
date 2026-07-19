<template>
  <header class="flex shrink-0 flex-wrap items-center gap-3 border-b border-slate-200 px-4 py-3 dark:border-white/10">
    <button
      class="rounded-lg border border-slate-200 p-2 lg:hidden dark:border-white/10"
      type="button"
      aria-label="Toggle sidebar"
      @click="$emit('toggle-sidebar')"
    >
      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/></svg>
    </button>

    <select
      :value="selectedProvider"
      class="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-white/10 dark:bg-slate-900"
      :disabled="disabled"
      @change="$emit('update:selectedProvider', ($event.target as HTMLSelectElement).value)"
    >
      <option v-for="name in providerNames" :key="name" :value="name">{{ name }}</option>
    </select>

    <select
      :value="selectedModel"
      class="min-w-0 flex-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm sm:flex-none dark:border-white/10 dark:bg-slate-900"
      :disabled="disabled"
      @change="$emit('update:selectedModel', ($event.target as HTMLSelectElement).value)"
    >
      <option v-for="model in models" :key="model.id" :value="model.id">{{ model.label || model.id }}{{ model.supports_tools ? ' 🔧' : '' }}</option>
    </select>

    <span v-if="supportsTools" class="hidden items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-semibold text-amber-700 sm:flex dark:bg-amber-900/30 dark:text-amber-300">
      🔧 Tools
    </span>

    <label
      v-if="toolsEnabled"
      class="flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-xs font-semibold"
      :class="yoloMode ? 'border-red-300 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-300' : 'border-slate-200 text-slate-600 dark:border-white/10 dark:text-slate-300'"
      title="When enabled, tool calls run without asking for approval"
    >
      <input :checked="yoloMode" type="checkbox" class="accent-red-600" :disabled="disabled" @change="$emit('update:yoloMode', ($event.target as HTMLInputElement).checked)" />
      YOLO mode
    </label>

    <span v-if="title" class="ml-auto hidden truncate text-xs text-slate-500 sm:block dark:text-slate-400">
      {{ title }}
    </span>
  </header>
</template>

<script setup lang="ts">
interface ModelInfo {
  id: string
  label?: string
  default?: boolean
  supports_tools?: boolean
}

defineProps<{
  providerNames: string[]
  selectedProvider: string
  selectedModel: string
  models: ModelInfo[]
  supportsTools: boolean
  toolsEnabled: boolean
  yoloMode: boolean
  disabled: boolean
  title?: string
}>()

defineEmits<{
  'toggle-sidebar': []
  'update:selectedProvider': [value: string]
  'update:selectedModel': [value: string]
  'update:yoloMode': [value: boolean]
}>()
</script>
