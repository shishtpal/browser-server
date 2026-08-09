<template>
  <header
    class="relative z-20 flex shrink-0 flex-wrap items-center gap-2 border-b border-slate-200/80 bg-white/80 px-2.5 py-2 backdrop-blur-md sm:px-3 dark:border-white/10 dark:bg-slate-950/70"
  >
    <!-- Sidebar toggle (mobile) -->
    <button
      class="grid h-9 w-9 place-items-center rounded-lg border border-slate-200 text-slate-500 transition-all hover:bg-slate-50 hover:text-slate-700 active:scale-95 lg:hidden dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-slate-200"
      type="button"
      aria-label="Open conversation list"
      @click="$emit('toggle-sidebar')"
    >
      <PanelLeft class="h-4 w-4" :stroke-width="2" aria-hidden="true" />
    </button>

    <!-- Profile selector -->
    <SearchableSelect
      v-if="profiles.length > 0"
      :model-value="selectedProfile"
      :items="profileItems"
      :disabled="disabled || profileLocked"
      :title="
        profileLocked ? 'Profile is locked for this conversation' : 'Select a system prompt profile'
      "
      class="w-32"
      @update:model-value="$emit('update:selectedProfile', $event)"
    >
      <template #trigger="{ label }">
        <div class="flex min-w-0 flex-1 items-center gap-1.5">
          <Lock
            v-if="profileLocked"
            class="h-3 w-3 shrink-0 text-amber-500"
            :stroke-width="2.25"
            aria-hidden="true"
          />
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
        <span class="flex min-w-0 flex-1 items-center gap-1 break-words whitespace-normal">
          <span class="min-w-0 flex-1">{{ item.label }}</span>
          <Wrench
            v-if="item.supports_tools"
            class="h-3 w-3 shrink-0 opacity-60"
            :stroke-width="2.25"
            aria-label="Supports tools"
          />
        </span>
      </template>
    </SearchableSelect>

    <!-- Tools badge -->
    <span
      v-if="supportsTools"
      class="hidden items-center gap-1 rounded-full bg-amber-50 px-2 py-0.5 text-[0.65rem] font-semibold tracking-wider text-amber-700 uppercase ring-1 ring-amber-200 ring-inset sm:inline-flex dark:bg-amber-900/20 dark:text-amber-300 dark:ring-amber-800/50"
    >
      <Wrench class="h-3 w-3" :stroke-width="2.25" aria-hidden="true" />
      Tools
    </span>

    <!-- YOLO toggle -->
    <label
      v-if="toolsEnabled"
      class="flex h-7 cursor-pointer items-center gap-1.5 rounded-lg border px-2.5 text-[0.72rem] font-semibold transition-colors"
      :class="
        yoloMode
          ? 'border-red-200 bg-red-50 text-red-700 ring-1 ring-red-200 ring-inset dark:border-red-900 dark:bg-red-950/30 dark:text-red-300 dark:ring-red-900/50'
          : 'border-slate-200 text-slate-500 hover:text-slate-700 dark:border-white/10 dark:text-slate-400 dark:hover:text-slate-300'
      "
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
        :class="
          activeSkills.includes(skill.name)
            ? 'border-emerald-200 bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 ring-inset dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300 dark:ring-emerald-800/50'
            : 'border-slate-200 text-slate-500 hover:border-slate-300 hover:text-slate-700 dark:border-white/10 dark:text-slate-400 dark:hover:border-white/20 dark:hover:text-slate-300'
        "
        :title="skill.description || skill.label"
        :aria-pressed="activeSkills.includes(skill.name)"
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
      <button
        :class="iconButtonClass(false)"
        type="button"
        title="Download conversation as Markdown"
        aria-label="Download conversation as Markdown"
        :disabled="downloadDisabled"
        @click="$emit('download')"
      >
        <Download class="h-4 w-4" :stroke-width="2" aria-hidden="true" />
      </button>

      <button
        v-for="action in toggles"
        :key="action.name"
        :class="[
          iconButtonClass(action.active),
          action.desktopOnly ? 'hidden lg:inline-flex' : 'inline-flex',
        ]"
        type="button"
        :title="action.title"
        :aria-label="action.title"
        :aria-pressed="action.active"
        @click="emitToggle(action.event)"
      >
        <component :is="action.icon" class="h-4 w-4" :stroke-width="2" aria-hidden="true" />
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import type { AIProfile, AISkill } from '@browser-server/shared-types';
import type { SelectItem } from '../ui/SearchableSelect.vue';
import { computed } from 'vue';
import {
  Brain,
  Download,
  Images,
  ListOrdered,
  Lock,
  PanelLeft,
  PanelRight,
  Wrench,
  type LucideIcon,
} from '@lucide/vue';
import SearchableSelect from '../ui/SearchableSelect.vue';

interface ModelInfo {
  id: string;
  label?: string;
  default?: boolean;
  supports_tools?: boolean;
}

const props = defineProps<{
  profiles: AIProfile[];
  selectedProfile: string;
  profileLocked: boolean;
  skills: AISkill[];
  activeSkills: string[];
  providerNames: string[];
  selectedProvider: string;
  selectedModel: string;
  models: ModelInfo[];
  supportsTools: boolean;
  toolsEnabled: boolean;
  yoloMode: boolean;
  disabled: boolean;
  title?: string;
  downloadDisabled?: boolean;
  showToolsPanel?: boolean;
  showMemoryExplorer?: boolean;
  showPromptManager?: boolean;
  showAttachmentGallery?: boolean;
}>();

const emit = defineEmits<{
  'toggle-sidebar': [];
  'update:selectedProfile': [value: string];
  'update:selectedProvider': [value: string];
  'update:selectedModel': [value: string];
  'update:yoloMode': [value: boolean];
  'toggle-skill': [name: string];
  download: [];
  'toggle-tools-panel': [];
  'toggle-memory-explorer': [];
  'toggle-prompt-manager': [];
  'toggle-attachment-gallery': [];
}>();

const emitToggle = (event: ToggleEvent) => (emit as any)(event);

const profileItems = computed<SelectItem[]>(() => [
  { value: '', label: 'Default' },
  ...props.profiles.map((p) => ({ value: p.name, label: p.label })),
]);

const providerItems = computed<SelectItem[]>(() =>
  props.providerNames.map((name) => ({ value: name, label: name })),
);

const modelItems = computed<SelectItem[]>(() =>
  props.models.map((m) => ({
    value: m.id,
    label: m.label || m.id,
    supports_tools: m.supports_tools,
  })),
);

type ToggleEvent =
  | 'toggle-memory-explorer'
  | 'toggle-attachment-gallery'
  | 'toggle-prompt-manager'
  | 'toggle-tools-panel';

const toggles = computed<
  Array<{
    name: string;
    title: string;
    icon: LucideIcon;
    active: boolean;
    event: ToggleEvent;
    desktopOnly?: boolean;
  }>
>(() => [
  {
    name: 'memory',
    title: 'Memory explorer',
    icon: Brain,
    active: props.showMemoryExplorer ?? false,
    event: 'toggle-memory-explorer',
  },
  {
    name: 'attachments',
    title: 'Attachment library',
    icon: Images,
    active: props.showAttachmentGallery ?? false,
    event: 'toggle-attachment-gallery',
  },
  {
    name: 'prompts',
    title: 'Prompt manager',
    icon: ListOrdered,
    active: props.showPromptManager ?? false,
    event: 'toggle-prompt-manager',
  },
  {
    name: 'tools',
    title: 'Toggle tools panel',
    icon: PanelRight,
    active: props.showToolsPanel ?? false,
    event: 'toggle-tools-panel',
    desktopOnly: true,
  },
]);

const iconButtonClass = computed(() => (active: boolean) => [
  'inline-flex h-9 w-9 items-center justify-center rounded-lg border transition-all active:scale-95 sm:h-auto sm:w-auto sm:p-1.5',
  active
    ? 'border-violet-200 bg-violet-50 text-violet-700 ring-1 ring-violet-200 ring-inset dark:border-violet-800 dark:bg-violet-950/30 dark:text-violet-300 dark:ring-violet-800/50'
    : 'border-slate-200 text-slate-500 hover:bg-slate-50 hover:text-slate-700 dark:border-white/10 dark:text-slate-400 dark:hover:bg-white/5 dark:hover:text-slate-200',
]);
</script>
