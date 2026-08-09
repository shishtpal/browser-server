<template>
  <aside
    ref="panelRef"
    class="relative flex h-full min-h-0 shrink-0 flex-col overflow-hidden border-l border-slate-200 bg-slate-50/80 dark:border-white/10 dark:bg-slate-900/60"
    :style="{ width: panelWidth + 'px' }"
  >
    <!-- Resize handle -->
    <div
      class="absolute inset-y-0 left-0 z-10 w-1.5 cursor-col-resize transition-colors select-none hover:bg-indigo-400/40 active:bg-indigo-500/50"
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize tools panel"
      @mousedown="startResize"
    ></div>

    <!-- Header -->
    <div
      class="flex shrink-0 items-center justify-between border-b border-slate-200 px-4 py-2.5 dark:border-white/10"
    >
      <h2 class="flex items-center gap-1.5 text-sm font-black">
        <PanelRight class="h-4 w-4 text-slate-400" :stroke-width="2.25" aria-hidden="true" />
        Panel
      </h2>
      <button
        class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-white/10 dark:hover:text-white"
        type="button"
        title="Close panel"
        aria-label="Close tools panel"
        @click="$emit('close')"
      >
        <X class="h-4 w-4" :stroke-width="2.5" aria-hidden="true" />
      </button>
    </div>

    <!-- Tabs -->
    <nav class="flex shrink-0 border-b border-slate-200 dark:border-white/10" role="tablist">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        role="tab"
        type="button"
        :aria-selected="activeTab === tab.id"
        :aria-controls="`chat-panel-${tab.id}`"
        class="relative flex-1 px-3 py-2.5 text-[11px] font-bold tracking-wide transition-colors"
        :class="
          activeTab === tab.id
            ? 'text-indigo-600 dark:text-indigo-400'
            : 'text-slate-400 hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300'
        "
        @click="activeTab = tab.id"
      >
        <span class="flex items-center justify-center gap-1.5">
          <component :is="tab.icon" class="h-3.5 w-3.5" :stroke-width="2.25" aria-hidden="true" />
          <span>{{ tab.label }}</span>
          <span
            v-if="tab.id === 'history' && toolCalls.length > 0"
            class="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-indigo-100 px-1 text-[9px] font-bold text-indigo-700 tabular-nums dark:bg-indigo-900/40 dark:text-indigo-300"
          >
            {{ toolCalls.length }}
          </span>
        </span>
        <span
          v-if="activeTab === tab.id"
          class="absolute inset-x-3 bottom-0 h-0.5 rounded-full bg-indigo-600 dark:bg-indigo-400"
          aria-hidden="true"
        ></span>
      </button>
    </nav>

    <!-- Tab content -->
    <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain">
      <div v-show="activeTab === 'tools'" id="chat-panel-tools" role="tabpanel">
        <ToolsTab
          :tools-enabled="toolsEnabled"
          :model-supports-tools="modelSupportsTools"
          :yolo-mode="yoloMode"
          :include-all-tool-definitions="includeAllToolDefinitions"
          :available-tools="availableTools"
          :tools-by-category="toolsByCategory"
          :disabled-tools="disabledTools"
          :mcp="mcp"
          :raw-tool-output="rawToolOutput"
          @update:tools-enabled="$emit('update:toolsEnabled', $event)"
          @update:yolo-mode="$emit('update:yoloMode', $event)"
          @update:include-all-tool-definitions="$emit('update:includeAllToolDefinitions', $event)"
          @update:raw-tool-output="$emit('update:rawToolOutput', $event)"
          @toggle-tool="(name, enabled) => $emit('toggle-tool', name, enabled)"
        />
      </div>

      <div v-show="activeTab === 'history'" id="chat-panel-history" role="tabpanel">
        <HistoryTab :tool-calls="toolCalls" />
      </div>

      <div v-show="activeTab === 'settings'" id="chat-panel-settings" role="tabpanel">
        <SettingsTab
          :font-family="fontFamily"
          :font-size="fontSize"
          v-model:show-thinking="showThinking"
          @update:font-family="$emit('update:fontFamily', $event)"
          @update:font-size="$emit('update:fontSize', $event)"
        />
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import type { AIMCPConfig } from '@browser-server/shared-types';
import type { ToolCallEntry } from './messages/messageTools';
import { ref } from 'vue';
import { useLocalStorage } from '@vueuse/core';
import { History, PanelRight, SlidersHorizontal, Wrench, X, type LucideIcon } from '@lucide/vue';
import ToolsTab from './panel/ToolsTab.vue';
import HistoryTab from './panel/HistoryTab.vue';
import SettingsTab from './panel/SettingsTab.vue';
import { usePanelResize } from './panel/usePanelResize';

defineProps<{
  toolsEnabled: boolean;
  modelSupportsTools: boolean;
  yoloMode: boolean;
  includeAllToolDefinitions: boolean;
  availableTools: string[];
  toolsByCategory: { category: string; tools: string[] }[];
  disabledTools: Set<string>;
  toolCalls: ToolCallEntry[];
  mcp: AIMCPConfig;
  fontFamily: string;
  fontSize: number;
  /** true = force raw, false = force JSON, null = follow server config */
  rawToolOutput: boolean | null;
}>();

defineEmits<{
  close: [];
  'update:toolsEnabled': [value: boolean];
  'update:yoloMode': [value: boolean];
  'update:includeAllToolDefinitions': [value: boolean];
  'update:fontFamily': [value: string];
  'update:fontSize': [value: number];
  'update:rawToolOutput': [value: boolean | null];
  'toggle-tool': [name: string, enabled: boolean];
}>();

const showThinking = defineModel<boolean>('showThinking', { default: true });

const panelRef = ref<HTMLElement | null>(null);

type TabId = 'tools' | 'history' | 'settings';

const tabs: { id: TabId; label: string; icon: LucideIcon }[] = [
  { id: 'tools', label: 'Tools', icon: Wrench },
  { id: 'history', label: 'History', icon: History },
  { id: 'settings', label: 'Settings', icon: SlidersHorizontal },
];

const activeTab = useLocalStorage<TabId>('ai-tools-panel-tab', 'tools');

const { panelWidth, startResize } = usePanelResize({
  storageKey: 'ai-tools-panel-width',
  min: 200,
  max: 500,
  initial: 280,
});
</script>
