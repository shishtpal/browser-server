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

    <!-- Skills toggles -->
    <div v-if="skills.length > 0" class="flex flex-wrap items-center gap-1.5">
      <button
        v-for="skill in skills"
        :key="skill.name"
        type="button"
        class="rounded-full border px-2.5 py-1 text-[11px] font-medium transition"
        :class="activeSkills.includes(skill.name)
          ? 'border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
          : 'border-slate-200 text-slate-500 hover:border-slate-300 hover:text-slate-700 dark:border-white/10 dark:text-slate-400 dark:hover:border-white/20 dark:hover:text-slate-300'"
        :title="skill.description || skill.label"
        :disabled="disabled"
        @click="$emit('toggle-skill', skill.name)"
      >
        {{ skill.label }}
      </button>
    </div>

    <span v-if="title" class="ml-auto hidden truncate text-xs text-slate-500 sm:block dark:text-slate-400">
      {{ title }}
    </span>

    <button
      class="rounded-lg border border-slate-200 p-2 text-slate-500 transition hover:bg-slate-50 hover:text-slate-700 disabled:cursor-not-allowed disabled:opacity-40 dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-slate-200"
      type="button"
      title="Download conversation as Markdown"
      aria-label="Download conversation as Markdown"
      :disabled="downloadDisabled"
      @click="$emit('download')"
    >
      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v12m0 0 4-4m-4 4-4-4M5 21h14a2 2 0 002-2v-3M3 16v3a2 2 0 002 2"/></svg>
    </button>

    <!-- Memory explorer toggle -->
    <button
      class="rounded-lg border p-2 transition"
      :class="showMemoryExplorer
        ? 'border-violet-300 bg-violet-50 text-violet-700 dark:border-violet-700 dark:bg-violet-950/40 dark:text-violet-300'
        : 'border-slate-200 text-slate-500 hover:bg-slate-50 dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5'"
      type="button"
      title="Memory Explorer"
      @click="$emit('toggle-memory-explorer')"
    >
      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
    </button>

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
import type { AIProfile, AISkill } from '@browser-server/shared-types'

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
  skills: AISkill[]
  activeSkills: string[]
  providerNames: string[]
  selectedProvider: string
  selectedModel: string
  models: ModelInfo[]
  supportsTools: boolean
  toolsEnabled: boolean
  yoloMode: boolean
  disabled: boolean
  title?: string
  downloadDisabled?: boolean
  showToolsPanel?: boolean
  showMemoryExplorer?: boolean
}>()

defineEmits<{
  'toggle-sidebar': []
  'update:selectedProfile': [value: string]
  'update:selectedProvider': [value: string]
  'update:selectedModel': [value: string]
  'update:yoloMode': [value: boolean]
  'toggle-skill': [name: string]
  download: []
  'toggle-tools-panel': []
  'toggle-memory-explorer': []
}>()
</script>
