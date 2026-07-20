<template>
  <aside
    ref="panelRef"
    class="relative flex min-h-0 h-full flex-col border-l border-slate-200 bg-slate-50/80 dark:border-white/10 dark:bg-slate-900/60"
    :style="{ width: panelWidth + 'px' }"
  >
    <!-- Resize handle -->
    <div
      class="absolute inset-y-0 left-0 z-10 w-1.5 cursor-col-resize select-none transition-colors hover:bg-indigo-400/40 active:bg-indigo-500/50"
      @mousedown="startResize"
    ></div>

    <!-- Header -->
    <div class="flex shrink-0 items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-white/10">
      <h2 class="text-sm font-black">Tools</h2>
      <button
        class="rounded-lg p-1.5 text-slate-400 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-white"
        type="button"
        title="Close panel"
        @click="$emit('close')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
      </button>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto p-4 space-y-5">
      <!-- Typography settings -->
      <section>
        <h3 class="mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">Typography</h3>
        <div class="space-y-2">
          <div>
            <label class="mb-1 block text-[10px] font-semibold text-slate-600 dark:text-slate-400">Font Family</label>
            <select
              :value="fontFamily"
              class="w-full rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs dark:border-white/10 dark:bg-slate-900"
              @change="$emit('update:fontFamily', ($event.target as HTMLSelectElement).value)"
            >
              <option value="system-ui">System Default</option>
              <option value="Inter, sans-serif">Inter</option>
              <option value="'JetBrains Mono', monospace">JetBrains Mono</option>
              <option value="'Fira Code', monospace">Fira Code</option>
              <option value="Georgia, serif">Georgia</option>
              <option value="Menlo, Monaco, monospace">Menlo / Monaco</option>
            </select>
          </div>
          <div>
            <label class="mb-1 block text-[10px] font-semibold text-slate-600 dark:text-slate-400">Font Size</label>
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
              <span class="w-8 text-center text-[10px] font-bold text-slate-600 dark:text-slate-400">{{ fontSize }}px</span>
            </div>
          </div>
        </div>
      </section>

      <hr class="border-slate-200 dark:border-white/10" />

      <!-- Tools toggle -->
      <section>
        <label class="flex items-center justify-between gap-3">
          <span class="text-xs font-bold text-slate-700 dark:text-slate-300">Enable tools</span>
          <input
            type="checkbox"
            :checked="toolsEnabled"
            class="h-4 w-4 accent-indigo-600"
            :disabled="!modelSupportsTools"
            @change="$emit('update:toolsEnabled', ($event.target as HTMLInputElement).checked)"
          />
        </label>
        <p v-if="!modelSupportsTools" class="mt-1 text-[10px] text-slate-400 dark:text-slate-500">
          Selected model does not support tools.
        </p>
      </section>

      <!-- YOLO mode -->
      <section v-if="toolsEnabled">
        <label
          class="flex items-center justify-between gap-3 rounded-lg border px-3 py-2"
          :class="yoloMode
            ? 'border-red-300 bg-red-50 dark:border-red-800 dark:bg-red-950/40'
            : 'border-slate-200 dark:border-white/10'"
        >
          <div>
            <span class="text-xs font-bold" :class="yoloMode ? 'text-red-700 dark:text-red-300' : 'text-slate-700 dark:text-slate-300'">YOLO mode</span>
            <p class="text-[10px] text-slate-400 dark:text-slate-500">Auto-approve all tool calls</p>
          </div>
          <input
            type="checkbox"
            :checked="yoloMode"
            class="h-4 w-4 accent-red-600"
            @change="$emit('update:yoloMode', ($event.target as HTMLInputElement).checked)"
          />
        </label>
      </section>

      <!-- Available tools -->
      <section v-if="toolsEnabled && availableTools.length > 0">
        <h3 class="mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">Available Tools</h3>
        <div class="space-y-1.5">
          <label
            v-for="tool in availableTools"
            :key="tool"
            class="flex items-center gap-2.5 rounded-lg border border-slate-200 px-3 py-2 transition hover:bg-white dark:border-white/10 dark:hover:bg-white/5"
          >
            <input
              type="checkbox"
              :checked="!disabledTools.has(tool)"
              class="h-3.5 w-3.5 accent-indigo-600"
              @change="$emit('toggle-tool', tool, ($event.target as HTMLInputElement).checked)"
            />
            <span class="flex-1 truncate text-xs font-semibold text-slate-700 dark:text-slate-300">{{ tool }}</span>
            <span class="grid h-5 w-5 place-items-center rounded bg-amber-100 text-[10px] dark:bg-amber-900/30">🔧</span>
          </label>
        </div>
      </section>

      <!-- Divider -->
      <hr v-if="toolCalls.length > 0" class="border-slate-200 dark:border-white/10" />

      <!-- Tool call history -->
      <section v-if="toolCalls.length > 0">
        <h3 class="mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
          Tool Calls ({{ toolCalls.length }})
        </h3>
        <div class="space-y-2">
          <div
            v-for="call in toolCalls"
            :key="call.id"
            class="rounded-lg border border-slate-200 bg-white p-2.5 dark:border-white/10 dark:bg-slate-900"
          >
            <div class="flex items-center gap-2">
              <span class="grid h-5 w-5 shrink-0 place-items-center rounded bg-amber-100 text-[10px] dark:bg-amber-900/30">🔧</span>
              <span class="flex-1 truncate text-xs font-bold text-slate-700 dark:text-slate-300">{{ call.name }}</span>
              <span
                class="shrink-0 rounded-full px-1.5 py-0.5 text-[9px] font-semibold"
                :class="statusClass(call.status)"
              >{{ call.status }}</span>
            </div>
            <details v-if="call.args" class="mt-1.5">
              <summary class="cursor-pointer text-[10px] font-medium text-slate-500 dark:text-slate-400">Args</summary>
              <pre class="mt-1 max-h-20 overflow-auto rounded bg-slate-100 p-1.5 text-[10px] leading-tight dark:bg-slate-800">{{ call.args }}</pre>
            </details>
            <details v-if="call.result" class="mt-1">
              <summary class="cursor-pointer text-[10px] font-medium text-slate-500 dark:text-slate-400">Result</summary>
              <pre class="mt-1 max-h-20 overflow-auto rounded bg-slate-100 p-1.5 text-[10px] leading-tight dark:bg-slate-800">{{ call.result }}</pre>
            </details>
          </div>
        </div>
      </section>

      <!-- Empty state when tools enabled but no calls yet -->
      <div v-if="toolsEnabled && toolCalls.length === 0" class="text-center text-xs text-slate-400 dark:text-slate-500 pt-4">
        <p>No tool calls in this conversation yet.</p>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { onUnmounted, ref } from 'vue'

const MIN_WIDTH = 200
const MAX_WIDTH = 500
const DEFAULT_WIDTH = 280
const STORAGE_KEY = 'ai-tools-panel-width'

export interface ToolCallEntry {
  id: string
  name: string
  status: string
  args?: string
  result?: string
}

defineProps<{
  toolsEnabled: boolean
  modelSupportsTools: boolean
  yoloMode: boolean
  availableTools: string[]
  disabledTools: Set<string>
  toolCalls: ToolCallEntry[]
  fontFamily: string
  fontSize: number
}>()

defineEmits<{
  close: []
  'update:toolsEnabled': [value: boolean]
  'update:yoloMode': [value: boolean]
  'update:fontFamily': [value: string]
  'update:fontSize': [value: number]
  'toggle-tool': [name: string, enabled: boolean]
}>()

// ─── Resize logic ──────────────────────────────────────

const panelRef = ref<HTMLElement | null>(null)
const panelWidth = ref(loadWidth())

let startX = 0
let startWidth = 0

function loadWidth(): number {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored) {
    const parsed = Number(stored)
    if (parsed >= MIN_WIDTH && parsed <= MAX_WIDTH) return parsed
  }
  return DEFAULT_WIDTH
}

function startResize(e: MouseEvent) {
  e.preventDefault()
  startX = e.clientX
  startWidth = panelWidth.value
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

function onResize(e: MouseEvent) {
  const delta = startX - e.clientX
  const newWidth = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, startWidth + delta))
  panelWidth.value = newWidth
}

function stopResize() {
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  localStorage.setItem(STORAGE_KEY, String(panelWidth.value))
}

onUnmounted(() => {
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
})

// ─── Helpers ───────────────────────────────────────────

function statusClass(status: string): string {
  switch (status) {
    case 'completed':
    case 'success':
      return 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'commented':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'error':
    case 'rejected':
      return 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400'
    case 'pending':
    case 'running':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
    default:
      return 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300'
  }
}
</script>
