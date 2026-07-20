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

    <!-- Profile selector -->
    <select
      v-if="profiles.length > 0"
      :value="selectedProfile"
      class="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-white/10 dark:bg-slate-900"
      :disabled="disabled || profileLocked"
      :title="profileLocked ? 'Profile is locked for this conversation' : 'Select a system prompt profile'"
      @change="$emit('update:selectedProfile', ($event.target as HTMLSelectElement).value)"
    >
      <option value="">Default</option>
      <option v-for="profile in profiles" :key="profile.name" :value="profile.name">{{ profile.label }}</option>
    </select>

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

    <!-- Tools panel toggle -->
    <button
      class="hidden rounded-lg border p-2 transition lg:block"
      :class="showToolsPanel
        ? 'border-indigo-300 bg-indigo-50 text-indigo-700 dark:border-indigo-700 dark:bg-indigo-950/40 dark:text-indigo-300'
        : 'border-slate-200 text-slate-500 hover:bg-slate-50 dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5'"
      type="button"
      title="Toggle tools panel"
      @click="$emit('toggle-tools-panel')"
    >
      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
    </button>
  </header>
</template>

<script setup lang="ts">
import type { AIProfile } from '@browser-server/shared-types'

interface ModelInfo {
  id: string
  label?: string
  default?: boolean
  supports_tools?: boolean
}

defineProps<{
  profiles: AIProfile[]
  selectedProfile: string
  profileLocked: boolean
  providerNames: string[]
  selectedProvider: string
  selectedModel: string
  models: ModelInfo[]
  supportsTools: boolean
  toolsEnabled: boolean
  yoloMode: boolean
  disabled: boolean
  title?: string
  showToolsPanel?: boolean
}>()

defineEmits<{
  'toggle-sidebar': []
  'update:selectedProfile': [value: string]
  'update:selectedProvider': [value: string]
  'update:selectedModel': [value: string]
  'update:yoloMode': [value: boolean]
  'toggle-tools-panel': []
}>()
</script>
