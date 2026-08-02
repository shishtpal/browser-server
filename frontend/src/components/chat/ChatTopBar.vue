<template>
  <header
    class="relative z-20 flex shrink-0 flex-wrap items-center gap-2 border-b border-slate-200/80 bg-white/80 px-3 py-2 backdrop-blur-md dark:border-white/10 dark:bg-slate-950/70"
  >
    <!-- Sidebar toggle (mobile) -->
    <button
      class="inline-flex items-center justify-center rounded-lg border border-slate-200 p-1.5 text-slate-500 transition-all hover:bg-slate-50 hover:text-slate-700 active:scale-95 lg:hidden dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-slate-200"
      type="button"
      aria-label="Toggle sidebar"
      @click="$emit('toggle-sidebar')"
    >
      <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
      </svg>
    </button>

    <!-- Profile selector -->
    <SearchableSelect
      v-if="profiles.length > 0"
      :model-value="selectedProfile"
      :items="profileItems"
      :disabled="disabled || profileLocked"
      :title="profileLocked ? 'Profile is locked for this conversation' : 'Select a system prompt profile'"
      class="w-32"
      @update:model-value="$emit('update:selectedProfile', $event)"
    >
      <template #trigger="{ label }">
        <div class="flex min-w-0 flex-1 items-center gap-1.5">
          <svg
            v-if="profileLocked"
            class="h-3 w-3 shrink-0 text-amber-500"
            fill="none" stroke="currentColor" viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
          </svg>
          <span class="overflow-hidden text-ellipsis whitespace-nowrap">{{ label }}</span>
        </div>
      </template>
    </SearchableSelect>

    <!-- Provider selector -->
    <SearchableSelect
      :model-value="selectedProvider"
      :items="providerItems"
      :disabled="disabled"
      class="w-28"
      @update:model-value="$emit('update:selectedProvider', $event)"
    />

    <!-- Model selector (searchable) -->
    <SearchableSelect
      :model-value="selectedModel"
      :items="modelItems"
      :disabled="disabled"
      :searchable="true"
      search-placeholder="Search models..."
      placeholder="Select a model"
      class="min-w-0 flex-1 sm:w-48 sm:flex-none"
      @update:model-value="$emit('update:selectedModel', $event)"
    >
      <template #item="{ item }">
        <span class="min-w-0 flex-1 whitespace-normal break-words">{{ item.label }}</span>
        <span v-if="item.supports_tools" class="shrink-0 text-xs opacity-60" title="Supports tools">🔧</span>
      </template>
    </SearchableSelect>

    <!-- Tools badge -->
    <span
      v-if="supportsTools"
      class="hidden items-center gap-1 rounded-full bg-amber-50 px-2 py-0.5 text-[0.65rem] font-semibold uppercase tracking-wider text-amber-700 ring-1 ring-inset ring-amber-200 sm:inline-flex dark:bg-amber-900/20 dark:text-amber-300 dark:ring-amber-800/50"
    >
      🔧 Tools
    </span>

    <!-- YOLO toggle -->
    <label
      v-if="toolsEnabled"
      class="flex h-7 cursor-pointer items-center gap-1.5 rounded-lg border px-2.5 text-[0.72rem] font-semibold transition-colors"
      :class="yoloMode
        ? 'border-red-200 bg-red-50 text-red-700 ring-1 ring-inset ring-red-200 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300 dark:ring-red-900/50'
        : 'border-slate-200 text-slate-500 hover:text-slate-700 dark:border-white/10 dark:text-slate-400 dark:hover:text-slate-300'"
      title="When enabled, tool calls run without asking for approval"
    >
      <input
        :checked="yoloMode"
        type="checkbox"
        class="accent-red-600"
        :disabled="disabled"
        @change="$emit('update:yoloMode', ($event.target as HTMLInputElement).checked)"
      />
      YOLO
    </label>

    <!-- Skills toggles -->
    <div v-if="skills.length > 0" class="flex flex-wrap items-center gap-1">
      <button
        v-for="skill in skills"
        :key="skill.name"
        type="button"
        class="rounded-full border px-2.5 py-0.5 text-[0.68rem] font-medium transition-all active:scale-95"
        :class="activeSkills.includes(skill.name)
          ? 'border-emerald-200 bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-200 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300 dark:ring-emerald-800/50'
          : 'border-slate-200 text-slate-500 hover:border-slate-300 hover:text-slate-700 dark:border-white/10 dark:text-slate-400 dark:hover:border-white/20 dark:hover:text-slate-300'"
        :title="skill.description || skill.label"
        :disabled="disabled"
        @click="$emit('toggle-skill', skill.name)"
      >
        {{ skill.label }}
      </button>
    </div>

    <!-- Conversation title -->
    <span
      v-if="title"
      class="ml-auto hidden max-w-[22ch] truncate text-[0.72rem] font-medium text-slate-400 sm:block dark:text-slate-500"
    >
      {{ title }}
    </span>

    <!-- Action buttons -->
    <div class="ml-auto flex items-center gap-1 sm:ml-2">
      <!-- Download -->
      <button
        class="inline-flex items-center justify-center rounded-lg border border-slate-200 p-1.5 text-slate-500 transition-all hover:bg-slate-50 hover:text-slate-700 active:scale-95 disabled:cursor-not-allowed disabled:opacity-40 dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-slate-200"
        type="button"
        title="Download conversation as Markdown"
        aria-label="Download conversation as Markdown"
        :disabled="downloadDisabled"
        @click="$emit('download')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v12m0 0 4-4m-4 4-4-4M5 21h14a2 2 0 002-2v-3M3 16v3a2 2 0 002 2"/>
        </svg>
      </button>

      <!-- Memory explorer -->
      <button
        class="inline-flex items-center justify-center rounded-lg border p-1.5 transition-all active:scale-95"
        :class="showMemoryExplorer
          ? 'border-violet-200 bg-violet-50 text-violet-700 ring-1 ring-inset ring-violet-200 dark:border-violet-800 dark:bg-violet-950/30 dark:text-violet-300 dark:ring-violet-800/50'
          : 'border-slate-200 text-slate-500 hover:bg-slate-50 dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5'"
        type="button"
        title="Memory Explorer"
        @click="$emit('toggle-memory-explorer')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 00-2-2V5a2 2 0 002-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 002 2z"/>
        </svg>
      </button>

      <!-- Prompt manager -->
      <button
        class="inline-flex items-center justify-center rounded-lg border p-1.5 transition-all active:scale-95"
        :class="showPromptManager
          ? 'border-emerald-200 bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-200 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300 dark:ring-emerald-800/50'
          : 'border-slate-200 text-slate-500 hover:bg-slate-50 dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5'"
        type="button"
        title="Prompt manager"
        @click="$emit('toggle-prompt-manager')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16"/>
        </svg>
      </button>

      <!-- Tools panel -->
      <button
        class="hidden items-center justify-center rounded-lg border p-1.5 transition-all active:scale-95 lg:inline-flex"
        :class="showToolsPanel
          ? 'border-indigo-200 bg-indigo-50 text-indigo-700 ring-1 ring-inset ring-indigo-200 dark:border-indigo-800 dark:bg-indigo-950/30 dark:text-indigo-300 dark:ring-indigo-800/50'
          : 'border-slate-200 text-slate-500 hover:bg-slate-50 dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5'"
        type="button"
        title="Toggle tools panel"
        @click="$emit('toggle-tools-panel')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
        </svg>
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import type { AIProfile, AISkill } from '@browser-server/shared-types'
import { computed } from 'vue'
import type { SelectItem } from '../ui/SearchableSelect.vue'
import SearchableSelect from '../ui/SearchableSelect.vue'

interface ModelInfo {
  id: string
  label?: string
  default?: boolean
  supports_tools?: boolean
}

const props = defineProps<{
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
  showPromptManager?: boolean
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
  'toggle-prompt-manager': []
}>()

const profileItems = computed<SelectItem[]>(() => [
  { value: '', label: 'Default' },
  ...props.profiles.map(p => ({ value: p.name, label: p.label })),
])

const providerItems = computed<SelectItem[]>(() =>
  props.providerNames.map(name => ({ value: name, label: name }))
)

const modelItems = computed<SelectItem[]>(() =>
  props.models.map(m => ({
    value: m.id,
    label: m.label || m.id,
    supports_tools: m.supports_tools,
  }))
)
</script>
