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
            {{ m.label || m.id }}{{ m.supports_tools ? ' 🔧' : '' }}
          </option>
        </select>
      </div>

      <!-- Skills -->
      <div v-if="skills.length > 0">
        <label class="mb-1.5 block text-xs font-semibold text-slate-600 dark:text-slate-300">
          Skills
        </label>
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="skill in skills"
            :key="skill.name"
            type="button"
            class="rounded-full border px-2.5 py-1 text-[11px] font-medium transition"
            :class="
              form.skills.includes(skill.name)
                ? 'border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300'
                : 'border-slate-200 text-slate-500 hover:border-slate-300 hover:text-slate-700 dark:border-white/10 dark:text-slate-400 dark:hover:border-white/20 dark:hover:text-slate-300'
            "
            :title="skill.description || skill.label"
            @click="toggleSkill(skill.name)"
          >
            {{ skill.label }}
          </button>
        </div>
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
import Modal from '../ui/Modal.vue';

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

function toggleSkill(name: string) {
  const idx = form.skills.indexOf(name);
  if (idx >= 0) {
    form.skills.splice(idx, 1);
  } else {
    form.skills.push(name);
  }
}

function handleCreate() {
  emit('create', {
    provider: form.provider,
    model: form.model,
    profile: form.profile,
    skills: [...form.skills],
  });
}
</script>
