<template>
  <aside
    ref="panelRef"
    class="relative flex h-full min-h-0 shrink-0 flex-col overflow-hidden border-l border-slate-200 bg-slate-50/80 dark:border-white/10 dark:bg-slate-900/60"
    :style="{ width: panelWidth + 'px' }"
  >
    <!-- Resize handle -->
    <div
      class="absolute inset-y-0 left-0 z-10 w-1.5 cursor-col-resize transition-colors select-none hover:bg-indigo-400/40 active:bg-indigo-500/50"
      @mousedown="startResize"
    ></div>

    <!-- Header with close button -->
    <div
      class="flex shrink-0 items-center justify-between border-b border-slate-200 px-4 py-2.5 dark:border-white/10"
    >
      <h2 class="text-sm font-black">Panel</h2>
      <button
        class="rounded-lg p-1.5 text-slate-400 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-white"
        type="button"
        title="Close panel"
        @click="$emit('close')"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>

    <!-- Tab bar -->
    <nav class="flex shrink-0 border-b border-slate-200 dark:border-white/10" role="tablist">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        role="tab"
        :aria-selected="activeTab === tab.id"
        :aria-controls="`panel-${tab.id}`"
        class="relative flex-1 px-3 py-2.5 text-[11px] font-bold tracking-wide transition-colors"
        :class="
          activeTab === tab.id
            ? 'text-indigo-600 dark:text-indigo-400'
            : 'text-slate-400 hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300'
        "
        @click="activeTab = tab.id"
      >
        <span class="flex items-center justify-center gap-1.5">
          <span>{{ tab.icon }}</span>
          <span>{{ tab.label }}</span>
          <span
            v-if="tab.id === 'history' && toolCalls.length > 0"
            class="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-indigo-100 px-1 text-[9px] font-bold text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300"
            >{{ toolCalls.length }}</span
          >
        </span>
        <!-- Active indicator -->
        <span
          v-if="activeTab === tab.id"
          class="absolute inset-x-3 bottom-0 h-0.5 rounded-full bg-indigo-600 dark:bg-indigo-400"
        ></span>
      </button>
    </nav>

    <!-- Tab content -->
    <div class="min-h-0 flex-1 overflow-y-auto">
      <!-- ═══ Tools tab ═══ -->
      <div v-show="activeTab === 'tools'" id="panel-tools" role="tabpanel" class="space-y-5 p-4">
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

        <!-- MCP server status -->
        <section
          v-if="mcp.configured"
          class="rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-white/10 dark:bg-slate-900"
        >
          <div class="flex items-center justify-between gap-2">
            <span class="text-xs font-bold text-slate-700 dark:text-slate-300">MCP servers</span>
            <span class="text-[10px] text-slate-400 dark:text-slate-500"
              >{{ connectedMCPServers }} connected · {{ mcpToolCount }} tools</span
            >
          </div>
          <div v-if="unavailableMCPServers.length > 0" class="mt-2 space-y-1.5">
            <div
              v-for="server in unavailableMCPServers"
              :key="server.name"
              class="rounded-md bg-red-50 px-2 py-1.5 dark:bg-red-950/30"
            >
              <p class="text-[10px] font-bold text-red-700 dark:text-red-300">
                {{ server.name }} unavailable
              </p>
              <p
                v-if="server.error"
                class="mt-0.5 text-[10px] text-red-600/80 dark:text-red-400/80"
              >
                {{ server.error }}
              </p>
            </div>
          </div>
        </section>

        <!-- YOLO mode -->
        <section v-if="toolsEnabled">
          <label
            class="flex items-center justify-between gap-3 rounded-lg border px-3 py-2"
            :class="
              yoloMode
                ? 'border-red-300 bg-red-50 dark:border-red-800 dark:bg-red-950/40'
                : 'border-slate-200 dark:border-white/10'
            "
          >
            <div>
              <span
                class="text-xs font-bold"
                :class="
                  yoloMode ? 'text-red-700 dark:text-red-300' : 'text-slate-700 dark:text-slate-300'
                "
                >YOLO mode</span
              >
              <p class="text-[10px] text-slate-400 dark:text-slate-500">
                Auto-approve all tool calls
              </p>
            </div>
            <input
              type="checkbox"
              :checked="yoloMode"
              class="h-4 w-4 accent-red-600"
              @change="$emit('update:yoloMode', ($event.target as HTMLInputElement).checked)"
            />
          </label>
        </section>

        <!-- Tool definition loading -->
        <section v-if="toolsEnabled">
          <label
            class="flex items-center justify-between gap-3 rounded-lg border border-slate-200 px-3 py-2 dark:border-white/10"
          >
            <div>
              <span class="text-xs font-bold text-slate-700 dark:text-slate-300"
                >Include all tool definitions</span
              >
              <p class="text-[10px] text-slate-400 dark:text-slate-500">
                Otherwise only search_tool is sent initially
              </p>
            </div>
            <input
              type="checkbox"
              :checked="includeAllToolDefinitions"
              class="h-4 w-4 accent-indigo-600"
              @change="
                $emit(
                  'update:includeAllToolDefinitions',
                  ($event.target as HTMLInputElement).checked,
                )
              "
            />
          </label>
        </section>

        <!-- Tool output mode -->
        <section v-if="toolsEnabled">
          <span class="text-xs font-bold text-slate-700 dark:text-slate-300">Tool output mode</span>
          <div
            class="mt-1.5 grid grid-cols-3 gap-1 rounded-lg border border-slate-200 bg-white p-1 dark:border-white/10 dark:bg-slate-900"
          >
            <button
              type="button"
              class="rounded-md px-2 py-1 text-[10px] font-bold transition"
              :class="
                rawToolOutput === true
                  ? 'bg-indigo-600 text-white'
                  : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10'
              "
              @click="$emit('update:rawToolOutput', true)"
            >
              Raw
            </button>
            <button
              type="button"
              class="rounded-md px-2 py-1 text-[10px] font-bold transition"
              :class="
                rawToolOutput === null
                  ? 'bg-indigo-600 text-white'
                  : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10'
              "
              @click="$emit('update:rawToolOutput', null)"
            >
              Auto
            </button>
            <button
              type="button"
              class="rounded-md px-2 py-1 text-[10px] font-bold transition"
              :class="
                rawToolOutput === false
                  ? 'bg-indigo-600 text-white'
                  : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10'
              "
              @click="$emit('update:rawToolOutput', false)"
            >
              JSON
            </button>
          </div>
          <p class="mt-1 text-[10px] text-slate-400 dark:text-slate-500">
            Raw returns bare text; Auto follows the server config allowlist; JSON wraps results in
            an envelope.
          </p>
        </section>

        <!-- Available tools (grouped by category) -->
        <section v-if="toolsEnabled && availableTools.length > 0">
          <h3
            class="mb-2 text-[10px] font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            Available Tools
          </h3>
          <!-- Search input -->
          <div class="relative mb-2">
            <Search
              class="pointer-events-none absolute top-1/2 left-2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400 dark:text-slate-500"
              :aria-hidden="true"
            />
            <input
              v-model="toolSearchQuery"
              type="search"
              placeholder="Search tools…"
              aria-label="Search tools"
              class="w-full rounded-lg border border-slate-200 bg-white py-1.5 pr-7 pl-7 text-xs text-slate-700 placeholder-slate-400 focus:border-indigo-400 focus:ring-1 focus:ring-indigo-400 focus:outline-none dark:border-white/10 dark:bg-slate-900 dark:text-slate-300 dark:placeholder-slate-500 dark:focus:border-indigo-500 dark:focus:ring-indigo-500"
            />
            <button
              v-if="toolSearchQuery"
              type="button"
              class="absolute top-1/2 right-1.5 -translate-y-1/2 rounded p-0.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
              aria-label="Clear search"
              title="Clear search"
              @click="clearSearch"
            >
              <X class="h-3.5 w-3.5" />
            </button>
          </div>
          <!-- Tool list -->
          <div v-if="filteredToolsByCategory.length > 0" class="space-y-2.5">
            <div
              v-for="group in filteredToolsByCategory"
              :key="group.category"
              class="overflow-hidden rounded-lg border border-slate-200 dark:border-white/10"
            >
              <!-- Category header with bulk toggle -->
              <div class="flex items-center gap-2 bg-slate-100/80 px-3 py-1.5 dark:bg-slate-800/60">
                <input
                  type="checkbox"
                  :checked="isCategoryFullyEnabled(visibleToggleTools(group.tools))"
                  :indeterminate="isCategoryPartial(visibleToggleTools(group.tools))"
                  class="h-3.5 w-3.5 accent-indigo-600"
                  @change="
                    toggleCategory(
                      visibleToggleTools(group.tools),
                      ($event.target as HTMLInputElement).checked,
                    )
                  "
                />
                <button
                  type="button"
                  class="flex flex-1 items-center gap-1.5 text-left"
                  @click="toggleCollapse(group.category)"
                >
                  <svg
                    class="h-3 w-3 text-slate-400 transition-transform duration-150"
                    :class="{ '-rotate-90': collapsed.has(group.category) }"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M19 9l-7 7-7-7"
                    />
                  </svg>
                  <span class="text-[11px] font-bold text-slate-600 dark:text-slate-300">{{
                    group.category
                  }}</span>
                  <span class="text-[10px] text-slate-400 dark:text-slate-500"
                    >({{ group.tools.length }})</span
                  >
                </button>
              </div>
              <!-- Tools within category -->
              <div
                v-show="isCategoryVisible(group.category)"
                class="divide-y divide-slate-100 dark:divide-white/5"
              >
                <label
                  v-for="tool in group.tools"
                  :key="tool"
                  class="flex cursor-pointer items-center gap-2.5 px-3 py-2 transition hover:bg-white dark:hover:bg-white/5"
                >
                  <input
                    type="checkbox"
                    :checked="!disabledTools.has(tool)"
                    class="h-3.5 w-3.5 accent-indigo-600"
                    @change="
                      $emit('toggle-tool', tool, ($event.target as HTMLInputElement).checked)
                    "
                  />
                  <span
                    class="flex-1 truncate text-xs font-semibold text-slate-700 dark:text-slate-300"
                    >{{ tool }}</span
                  >
                  <span
                    class="grid h-5 w-5 place-items-center rounded bg-amber-100 text-[10px] dark:bg-amber-900/30"
                    >🔧</span
                  >
                </label>
              </div>
            </div>
          </div>
          <!-- No results -->
          <div
            v-if="toolsEnabled && availableTools.length > 0 && filteredToolsByCategory.length === 0"
            class="py-4 text-center"
          >
            <p class="text-xs text-slate-400 dark:text-slate-500">
              No tools match &quot;{{ toolSearchQuery }}&quot;.
            </p>
            <button
              type="button"
              class="mt-2 inline-flex items-center gap-1 rounded-md bg-slate-100 px-2.5 py-1 text-[10px] font-semibold text-slate-600 transition hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700"
              @click="clearSearch"
            >
              <X class="h-3 w-3" />
              Clear search
            </button>
          </div>
        </section>

        <!-- Empty state -->
        <div
          v-if="toolsEnabled && availableTools.length === 0"
          class="pt-4 text-center text-xs text-slate-400 dark:text-slate-500"
        >
          <p>No tools available for this model.</p>
        </div>
        <div
          v-if="!toolsEnabled"
          class="pt-4 text-center text-xs text-slate-400 dark:text-slate-500"
        >
          <p>Enable tools to configure available tool calls.</p>
        </div>
      </div>

      <!-- ═══ History tab ═══ -->
      <div
        v-show="activeTab === 'history'"
        id="panel-history"
        role="tabpanel"
        class="space-y-3 p-4"
      >
        <div v-if="toolCalls.length > 0" class="space-y-2">
          <div
            v-for="call in toolCalls"
            :key="call.id"
            class="rounded-lg border border-slate-200 bg-white p-2.5 dark:border-white/10 dark:bg-slate-900"
          >
            <div class="flex items-center gap-2">
              <span
                class="grid h-5 w-5 shrink-0 place-items-center rounded bg-amber-100 text-[10px] dark:bg-amber-900/30"
                >🔧</span
              >
              <span class="flex-1 truncate text-xs font-bold text-slate-700 dark:text-slate-300">{{
                call.name
              }}</span>
              <span
                class="shrink-0 rounded-full px-1.5 py-0.5 text-[9px] font-semibold"
                :class="statusClass(call.status)"
                >{{ call.status }}</span
              >
            </div>
            <details v-if="call.args" class="mt-1.5">
              <summary
                class="cursor-pointer text-[10px] font-medium text-slate-500 dark:text-slate-400"
              >
                Args
              </summary>
              <pre
                class="mt-1 max-h-20 overflow-auto rounded bg-slate-100 p-1.5 text-[10px] leading-tight dark:bg-slate-800"
                >{{ call.args }}</pre>
            </details>
            <details v-if="call.result" class="mt-1">
              <summary
                class="cursor-pointer text-[10px] font-medium text-slate-500 dark:text-slate-400"
              >
                Result
              </summary>
              <pre
                class="mt-1 max-h-20 overflow-auto rounded bg-slate-100 p-1.5 text-[10px] leading-tight dark:bg-slate-800"
                >{{ call.result }}</pre>
            </details>
          </div>
        </div>
        <!-- Empty state -->
        <div v-else class="pt-8 text-center text-xs text-slate-400 dark:text-slate-500">
          <span class="text-2xl">📋</span>
          <p class="mt-2">No tool calls in this conversation yet.</p>
        </div>
      </div>

      <!-- ═══ Settings tab ═══ -->
      <div
        v-show="activeTab === 'settings'"
        id="panel-settings"
        role="tabpanel"
        class="space-y-5 p-4"
      >
        <!-- Typography settings -->
        <section>
          <h3
            class="mb-2 text-[10px] font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            Typography
          </h3>
          <div class="space-y-2">
            <div>
              <label class="mb-1 block text-[10px] font-semibold text-slate-600 dark:text-slate-400"
                >Font Family</label
              >
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
              <label class="mb-1 block text-[10px] font-semibold text-slate-600 dark:text-slate-400"
                >Font Size</label
              >
              <div class="flex items-center gap-2">
                <input
                  type="range"
                  :value="fontSize"
                  min="12"
                  max="20"
                  step="1"
                  class="h-1.5 flex-1 cursor-pointer appearance-none rounded-full bg-slate-200 accent-indigo-600 dark:bg-slate-700"
                  @input="
                    $emit('update:fontSize', Number(($event.target as HTMLInputElement).value))
                  "
                />
                <span
                  class="w-8 text-center text-[10px] font-bold text-slate-600 dark:text-slate-400"
                  >{{ fontSize }}px</span
                >
              </div>
            </div>
          </div>
        </section>

        <!-- Reasoning display -->
        <section>
          <h3
            class="mb-2 text-[10px] font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
          >
            Thinking
          </h3>
          <label
            class="flex cursor-pointer items-start gap-2 text-xs text-slate-600 dark:text-slate-400"
          >
            <input v-model="showThinking" type="checkbox" class="mt-0.5 accent-indigo-600" />
            <span>
              <span class="font-semibold text-slate-700 dark:text-slate-300"
                >Show model thinking</span
              >
              <span class="mt-0.5 block text-[10px] text-slate-400 dark:text-slate-500"
                >Display reasoning when the model provides it.</span
              >
            </span>
          </label>
        </section>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import type { AIMCPConfig } from '@browser-server/shared-types'
import { computed, onUnmounted, reactive, ref } from 'vue'
import { Search, X } from '@lucide/vue'

const MIN_WIDTH = 200
const MAX_WIDTH = 500
const DEFAULT_WIDTH = 280
const STORAGE_KEY = 'ai-tools-panel-width'
const TAB_STORAGE_KEY = 'ai-tools-panel-tab'

export interface ToolCallEntry {
  id: string
  name: string
  status: string
  args?: string
  result?: string
}

type TabId = 'tools' | 'history' | 'settings'

const tabs: { id: TabId; label: string; icon: string }[] = [
  { id: 'tools', label: 'Tools', icon: '🔧' },
  { id: 'history', label: 'History', icon: '📋' },
  { id: 'settings', label: 'Settings', icon: '⚙️' },
]

const props = defineProps<{
  toolsEnabled: boolean
  modelSupportsTools: boolean
  yoloMode: boolean
  includeAllToolDefinitions: boolean
  availableTools: string[]
  toolsByCategory: { category: string; tools: string[] }[]
  disabledTools: Set<string>
  toolCalls: ToolCallEntry[]
  mcp: AIMCPConfig
  fontFamily: string
  fontSize: number
  /** true = force raw, false = force JSON, null = follow server config */
  rawToolOutput: boolean | null
}>()

const emit = defineEmits<{
  close: []
  'update:toolsEnabled': [value: boolean]
  'update:yoloMode': [value: boolean]
  'update:includeAllToolDefinitions': [value: boolean]
  'update:fontFamily': [value: string]
  'update:fontSize': [value: number]
  'update:rawToolOutput': [value: boolean | null]
  'toggle-tool': [name: string, enabled: boolean]
}>()

const showThinking = defineModel<boolean>('showThinking', {
  default: true,
})

// ─── Tab state ─────────────────────────────────────────

function loadTab(): TabId {
  const stored = localStorage.getItem(TAB_STORAGE_KEY) as TabId | null
  if (stored && tabs.some((t) => t.id === stored)) return stored
  return 'tools'
}

const activeTab = ref<TabId>(loadTab())

// Persist tab selection
import { watch } from 'vue'
watch(activeTab, (val) => {
  localStorage.setItem(TAB_STORAGE_KEY, val)
})

// ─── Category grouping logic ───────────────────────────

const toolSearchQuery = ref('')

const filteredToolsByCategory = computed(() => {
  const q = toolSearchQuery.value.toLowerCase().trim()
  if (!q) return props.toolsByCategory
  return props.toolsByCategory
    .map((g) => ({
      category: g.category,
      tools: g.tools.filter((t) => t.toLowerCase().includes(q)),
    }))
    .filter((g) => g.tools.length > 0)
})

const connectedMCPServers = computed(
  () => props.mcp.servers.filter((server) => server.status === 'connected').length,
)
const unavailableMCPServers = computed(() =>
  props.mcp.servers.filter((server) => server.status === 'unavailable'),
)
const mcpToolCount = computed(() =>
  props.mcp.servers.reduce((total, server) => total + (server.tools?.length ?? 0), 0),
)

function isCategoryVisible(category: string): boolean {
  if (!toolSearchQuery.value.trim()) return !collapsed.has(category)
  return true
}

function visibleToggleTools(tools: string[]): string[] {
  const q = toolSearchQuery.value.toLowerCase().trim()
  if (!q) return tools
  return tools.filter((t) => t.toLowerCase().includes(q))
}

function clearSearch() {
  toolSearchQuery.value = ''
}

const collapsed = reactive(new Set<string>())

function toggleCollapse(category: string) {
  if (collapsed.has(category)) {
    collapsed.delete(category)
  } else {
    collapsed.add(category)
  }
}

function isCategoryFullyEnabled(tools: string[]): boolean {
  return tools.every((t) => !props.disabledTools.has(t))
}

function isCategoryPartial(tools: string[]): boolean {
  const enabled = tools.filter((t) => !props.disabledTools.has(t)).length
  return enabled > 0 && enabled < tools.length
}

function toggleCategory(tools: string[], enabled: boolean) {
  for (const tool of tools) {
    emit('toggle-tool', tool, enabled)
  }
}

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
