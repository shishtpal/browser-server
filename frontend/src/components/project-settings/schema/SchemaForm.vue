<script setup lang="ts">
import { computed, ref } from 'vue';
import type { SchemaNode } from './schemaUtils';
import { fieldSchemaFor, humanizeKey, makeDefault } from './schemaUtils';
import SchemaField from './SchemaField.vue';

const props = withDefaults(
  defineProps<{
    schema?: SchemaNode;
    modelValue?: Record<string, unknown> | null;
    title?: string;
  }>(),
  { title: 'Fields' },
);
const emit = defineEmits<{
  (e: 'update:modelValue', value: Record<string, unknown>): void;
}>();

const data = computed<Record<string, unknown>>(() => {
  const v = props.modelValue;
  return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {};
});

const properties = computed(() => {
  const p = props.schema?.properties;
  return p && typeof p === 'object' ? p : {};
});

const requiredKeys = computed(() => new Set<string>(props.schema?.required ?? []));

// Schema keys that are not yet present in the data (for the add-field picker).
const missingKeys = computed(() => Object.keys(properties.value).filter((k) => !(k in data.value)));

const pendingKey = ref('');

function setValue(key: string, value: unknown) {
  emit('update:modelValue', { ...data.value, [key]: value });
}

function addField() {
  const key = pendingKey.value;
  if (!key || key in data.value) return;
  setValue(key, makeDefault(properties.value[key]));
  pendingKey.value = '';
}
</script>

<template>
  <div class="flex flex-col gap-2">
    <h3
      v-if="props.title"
      class="text-[11px] font-bold text-slate-500 uppercase dark:text-slate-400"
    >
      {{ props.title }}
    </h3>

    <div
      v-for="key in Object.keys(data)"
      :key="key"
      class="rounded-lg border border-slate-200 p-3 dark:border-white/10"
    >
      <SchemaField
        :name="key"
        :schema="fieldSchemaFor(props.schema, key)"
        :model-value="data[key]"
        :required="requiredKeys.has(key)"
        @update:model-value="setValue(key, $event)"
      />
    </div>

    <div v-if="missingKeys.length" class="flex items-end gap-2 pt-1">
      <label class="text-xs text-slate-500 dark:text-slate-400">Add field</label>
      <select
        v-model="pendingKey"
        class="flex-1 rounded border border-slate-300 bg-white text-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
      >
        <option value="" disabled>— choose a field —</option>
        <option v-for="key in missingKeys" :key="key" :value="key">
          {{ humanizeKey(key) }}
        </option>
      </select>
      <button
        type="button"
        :disabled="!pendingKey"
        @click="addField"
        class="rounded bg-violet-600 px-2 py-1 text-xs font-bold text-white hover:bg-violet-500 disabled:opacity-40"
      >
        Add
      </button>
    </div>

    <p
      v-if="!Object.keys(properties).length"
      class="text-[11px] text-slate-500 dark:text-slate-400"
    >
      This section has no schema-defined fields.
    </p>
  </div>
</template>
