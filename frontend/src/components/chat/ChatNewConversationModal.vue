<template>
  <Modal
    :open="open"
    title="New Conversation"
    description="Choose settings for the new conversation."
    @close="$emit('close')"
  >
    <form class="space-y-4" @submit.prevent="handleCreate">
      <!-- Profile -->
      <div v-if="profiles.length > 0">
        <label class="mb-1 block text-xs font-semibold text-slate-600 dark:text-slate-300">
          Profile
        </label>
        <select
          v-model="form.profile"
          class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 dark:border-white/10 dark:bg-slate-900 dark:text-white"
        >
          <option value="">Default</option>
          <option v-for="p in profiles" :key="p.name" :value="p.name">{{ p.label }}</option>
        </select>
      </div>

      <!-- Provider -->
      <div>
        <label class="mb-1 block text-xs font-semibold text-slate-600 dark:text-slate-300">
          Provider
        </label>
        <select
          v-model="form.provider"
          class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 dark:border-white/10 dark:bg-slate-900 dark:text-white"
        >
          <option v-for="name in providerNames" :key="name" :value="name">{{ name }}</option>
        </select>
      </div>

      <!-- Model -->
      <div>
        <label class="mb-1 block text-xs font-semibold text-slate-600 dark:text-slate-300">
          Model
        </label>
        <select
          v-model="form.model"
          class="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 dark:border-white/10 dark:bg-slate-900 dark:text-white"
        >
          <option v-for="m in currentModels" :key="m.id" :value="m.id">
            <span class="inline-flex items-center gap-1">
              {{ m.label || m.id }}
              <Wrench
                v-if="m.supports_tools"
                class="h-3 w-3 text-amber-500"
                :stroke-width="2.25"
                aria-label="Supports tools"
              />
            </span>
          </option>
        </select>
      </div>

      <!-- Skills -->
      <div v-if="skills.length > 0">
        <label class="mb-1 block text-xs font-semibold text-slate-600 dark:text-slate-300">
          Skills
        </label>
        <MultiSelectDropdown
          :model-value="form.skills"
          :items="skillItems"
          placeholder="Select skills..."
          :searchable="true"
          search-placeholder="Search skills..."
          class="w-full"
          @update:model-value="form.skills = $event"
        />
      </div>

      <!-- Actions -->
      <div class="flex justify-end gap-2 pt-2">
        <button
          class="rounded-lg px-4 py-2 text-sm font-bold text-slate-500 hover:bg-slate-100 dark:hover:bg-white/10"
          type="button"
          @click="$emit('close')"
        >
          Cancel
        </button>
        <button
          class="rounded-lg bg-slate-900 px-4 py-2 text-sm font-bold text-white transition hover:bg-slate-800 dark:bg-white dark:text-slate-900 dark:hover:bg-gray-100"
          type="submit"
        >
          Create
        </button>
      </div>
    </form>
  </Modal>
</template>

<script setup lang="ts">
import type { AIProfile, AISkill, AIProviderConfig } from '@browser-server/shared-types';
import { reactive, watch, computed } from 'vue';
import { Wrench } from '@lucide/vue';
import Modal from '../ui/Modal.vue';
import MultiSelectDropdown, { type SelectItem } from '../ui/MultiSelectDropdown.vue';

export interface NewConversationResult {
  provider: string;
  model: string;
  profile: string;
  skills: string[];
}

const props = defineProps<{
  open: boolean;
  profiles: AIProfile[];
  providerNames: string[];
  providers: Record<string, AIProviderConfig>;
  skills: AISkill[];
  defaultProvider: string;
  defaultModel: string;
  defaultProfile: string;
  defaultSkills: string[];
}>();

const emit = defineEmits<{
  close: [];
  create: [result: NewConversationResult];
}>();

const form = reactive({
  profile: '',
  provider: '',
  model: '',
  skills: [] as string[],
});

// Models for the currently selected provider in the form
const currentModels = computed(() => {
  const provider = props.providers[form.provider];
  return provider?.models ?? [];
});

// Reset form when modal opens
watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      form.profile = props.defaultProfile;
      form.provider = props.defaultProvider;
      form.model = props.defaultModel;
      form.skills = [...props.defaultSkills];
    }
  },
);

// When provider changes in the form, pick a reasonable default model
watch(
  () => form.provider,
  () => {
    const models = currentModels.value;
    if (models.length > 0 && !models.some((m) => m.id === form.model)) {
      form.model = models.find((m) => m.default)?.id || models[0]?.id || '';
    }
  },
);

const skillItems = computed<SelectItem[]>(() =>
  props.skills.map((s) => ({
    value: s.name,
    label: s.label,
    description: s.description,
  })),
);

function handleCreate() {
  emit('create', {
    provider: form.provider,
    model: form.model,
    profile: form.profile,
    skills: [...form.skills],
  });
}
</script>
