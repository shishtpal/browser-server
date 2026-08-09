<template>
  <div class="space-y-5 p-4">
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
        <span
          class="flex items-center gap-1.5 text-xs font-bold text-slate-700 dark:text-slate-300"
        >
          <Globe class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
          MCP servers
        </span>
        <span class="text-[10px] text-slate-400 dark:text-slate-500">
          {{ connectedCount }} connected · {{ toolCount }} tools
        </span>
      </div>
      <div v-if="unavailable.length > 0" class="mt-2 space-y-1.5">
        <div
          v-for="server in unavailable"
          :key="server.name"
          class="rounded-md bg-red-50 px-2 py-1.5 dark:bg-red-950/30"
        >
          <p class="text-[10px] font-bold text-red-700 dark:text-red-300">
            {{ server.name }} unavailable
          </p>
          <p v-if="server.error" class="mt-0.5 text-[10px] text-red-600/80 dark:text-red-400/80">
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
          >
            YOLO mode
          </span>
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

    <!-- Tool definition loading -->
    <section v-if="toolsEnabled">
      <label
        class="flex items-center justify-between gap-3 rounded-lg border border-slate-200 px-3 py-2 dark:border-white/10"
      >
        <div>
          <span class="text-xs font-bold text-slate-700 dark:text-slate-300">
            Include all tool definitions
          </span>
          <p class="text-[10px] text-slate-400 dark:text-slate-500">
            Otherwise only search_tool is sent initially
          </p>
        </div>
        <input
          type="checkbox"
          :checked="includeAllToolDefinitions"
          class="h-4 w-4 accent-indigo-600"
          @change="
            $emit('update:includeAllToolDefinitions', ($event.target as HTMLInputElement).checked)
          "
        />
      </label>
    </section>

    <!-- Tool output mode -->
    <section v-if="toolsEnabled">
      <span class="text-xs font-bold text-slate-700 dark:text-slate-300">Tool output mode</span>
      <div
        class="mt-1.5 grid grid-cols-3 gap-1 rounded-lg border border-slate-200 bg-white p-1 dark:border-white/10 dark:bg-slate-900"
        role="group"
        aria-label="Tool output mode"
      >
        <button
          v-for="opt in outputModes"
          :key="opt.label"
          type="button"
          class="rounded-md px-2 py-1 text-[10px] font-bold transition"
          :class="
            rawToolOutput === opt.value
              ? 'bg-indigo-600 text-white'
              : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-white/10'
          "
          :aria-pressed="rawToolOutput === opt.value"
          @click="$emit('update:rawToolOutput', opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>
      <p class="mt-1 text-[10px] text-slate-400 dark:text-slate-500">
        Raw returns bare text; Auto follows the server config allowlist; JSON wraps results in an
        envelope.
      </p>
    </section>

    <!-- Available tools -->
    <section v-if="toolsEnabled && availableTools.length > 0">
      <h3
        class="mb-2 text-[10px] font-bold tracking-wider text-slate-500 uppercase dark:text-slate-400"
      >
        Available Tools
      </h3>

      <!-- Search -->
      <div class="relative mb-2">
        <Search
          class="pointer-events-none absolute top-1/2 left-2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400 dark:text-slate-500"
          aria-hidden="true"
        />
        <input
          v-model="query"
          type="search"
          placeholder="Search tools…"
          aria-label="Search tools"
          class="w-full rounded-lg border border-slate-200 bg-white py-1.5 pr-7 pl-7 text-xs text-slate-700 placeholder-slate-400 focus:border-indigo-400 focus:ring-1 focus:ring-indigo-400 focus:outline-none dark:border-white/10 dark:bg-slate-900 dark:text-slate-300 dark:placeholder-slate-500 dark:focus:border-indigo-500 dark:focus:ring-indigo-500"
        />
        <button
          v-if="query"
          type="button"
          class="absolute top-1/2 right-1.5 -translate-y-1/2 rounded p-0.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
          aria-label="Clear search"
          title="Clear search"
          @click="query = ''"
        >
          <X class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
        </button>
      </div>

      <!-- Category groups -->
      <div v-if="filteredGroups.length > 0" class="space-y-2.5">
        <div
          v-for="group in filteredGroups"
          :key="group.category"
          class="overflow-hidden rounded-lg border border-slate-200 dark:border-white/10"
        >
          <div class="flex items-center gap-2 bg-slate-100/80 px-3 py-1.5 dark:bg-slate-800/60">
            <input
              type="checkbox"
              :checked="allEnabled(visibleTools(group.tools))"
              :indeterminate="partiallyEnabled(visibleTools(group.tools))"
              class="h-3.5 w-3.5 accent-indigo-600"
              :aria-label="`Toggle all ${group.category} tools`"
              @change="
                toggleCategory(
                  visibleTools(group.tools),
                  ($event.target as HTMLInputElement).checked,
                )
              "
            />
            <button
              type="button"
              class="flex min-w-0 flex-1 items-center gap-1.5 text-left"
              :aria-expanded="isVisible(group.category)"
              @click="toggleCollapse(group.category)"
            >
              <ChevronDown
                class="h-3 w-3 text-slate-400 transition-transform duration-150"
                :class="{ '-rotate-90': collapsed.has(group.category) }"
                aria-hidden="true"
              />
              <span class="truncate text-[11px] font-bold text-slate-600 dark:text-slate-300">
                {{ group.category }}
              </span>
              <span class="shrink-0 text-[10px] text-slate-400 dark:text-slate-500">
                ({{ group.tools.length }})
              </span>
            </button>
          </div>

          <div
            v-show="isVisible(group.category)"
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
                @change="$emit('toggle-tool', tool, ($event.target as HTMLInputElement).checked)"
              />
              <span
                class="min-w-0 flex-1 truncate text-xs font-semibold text-slate-700 dark:text-slate-300"
              >
                {{ tool }}
              </span>
              <Wrench
                class="h-3 w-3 shrink-0 text-amber-500/70"
                :stroke-width="2.25"
                aria-hidden="true"
              />
            </label>
          </div>
        </div>
      </div>

      <!-- No results -->
      <div v-if="filteredGroups.length === 0" class="py-4 text-center">
        <p class="text-xs text-slate-400 dark:text-slate-500">No tools match "{{ query }}".</p>
        <button
          type="button"
          class="mt-2 inline-flex items-center gap-1 rounded-md bg-slate-100 px-2.5 py-1 text-[10px] font-semibold text-slate-600 transition hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700"
          @click="query = ''"
        >
          <X class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
          Clear search
        </button>
      </div>
    </section>

    <!-- Empty states -->
    <div
      v-if="toolsEnabled && availableTools.length === 0"
      class="pt-4 text-center text-xs text-slate-400 dark:text-slate-500"
    >
      <p>No tools available for this model.</p>
    </div>
    <div v-if="!toolsEnabled" class="pt-4 text-center text-xs text-slate-400 dark:text-slate-500">
      <p>Enable tools to configure available tool calls.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { ChevronDown, Globe, Search, Wrench, X } from '@lucide/vue';
import type { AIMCPConfig } from '@browser-server/shared-types';

const props = defineProps<{
  toolsEnabled: boolean;
  modelSupportsTools: boolean;
  yoloMode: boolean;
  includeAllToolDefinitions: boolean;
  availableTools: string[];
  toolsByCategory: { category: string; tools: string[] }[];
  disabledTools: Set<string>;
  mcp: AIMCPConfig;
  /** true = force raw, false = force JSON, null = follow server config */
  rawToolOutput: boolean | null;
}>();

const emit = defineEmits<{
  'update:toolsEnabled': [value: boolean];
  'update:yoloMode': [value: boolean];
  'update:includeAllToolDefinitions': [value: boolean];
  'update:rawToolOutput': [value: boolean | null];
  'toggle-tool': [name: string, enabled: boolean];
}>();

const outputModes = [
  { label: 'Raw', value: true as boolean | null },
  { label: 'Auto', value: null as boolean | null },
  { label: 'JSON', value: false as boolean | null },
];

const query = ref('');
const collapsed = reactive(new Set<string>());

const filteredGroups = computed(() => {
  const q = query.value.toLowerCase().trim();
  if (!q) return props.toolsByCategory;
  return props.toolsByCategory
    .map((g) => ({
      category: g.category,
      tools: g.tools.filter((t) => t.toLowerCase().includes(q)),
    }))
    .filter((g) => g.tools.length > 0);
});

const connectedCount = computed(
  () => props.mcp.servers.filter((s) => s.status === 'connected').length,
);
const unavailable = computed(() => props.mcp.servers.filter((s) => s.status === 'unavailable'));
const toolCount = computed(() =>
  props.mcp.servers.reduce((total, s) => total + (s.tools?.length ?? 0), 0),
);

/** While searching, all matched categories stay expanded. */
function isVisible(category: string): boolean {
  if (!query.value.trim()) return !collapsed.has(category);
  return true;
}

function visibleTools(tools: string[]): string[] {
  const q = query.value.toLowerCase().trim();
  if (!q) return tools;
  return tools.filter((t) => t.toLowerCase().includes(q));
}

function toggleCollapse(category: string) {
  if (collapsed.has(category)) collapsed.delete(category);
  else collapsed.add(category);
}

function allEnabled(tools: string[]): boolean {
  return tools.every((t) => !props.disabledTools.has(t));
}

function partiallyEnabled(tools: string[]): boolean {
  const enabled = tools.filter((t) => !props.disabledTools.has(t)).length;
  return enabled > 0 && enabled < tools.length;
}

function toggleCategory(tools: string[], enabled: boolean) {
  for (const tool of tools) emit('toggle-tool', tool, enabled);
}
</script>
