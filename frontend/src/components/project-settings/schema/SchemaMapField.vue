<script setup lang="ts">
import { computed, ref } from 'vue';
import type { SchemaNode, SchemaValue } from './schemaUtils';
import { makeDefault } from './schemaUtils';
import SchemaField from './SchemaField.vue';

const props = defineProps<{
  schema?: SchemaNode;
  modelValue?: SchemaValue;
}>();
const emit = defineEmits<{ (e: 'update:modelValue', value: Record<string, unknown>): void }>();

const valueSchema = computed<SchemaNode | undefined>(() => {
  const ap = props.schema?.additionalProperties;
  if (ap && typeof ap === 'object' && !Array.isArray(ap)) return ap as SchemaNode;
  return undefined;
});

const record = computed<Record<string, unknown>>(() => {
  const v = props.modelValue;
  return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {};
});

const entries = computed(() => Object.keys(record.value));
const pendingKey = ref('');

function setEntry(key: string, value: unknown) {
  emit('update:modelValue', { ...record.value, [key]: value });
}

function renameKey(oldKey: string, newKey: string) {
  if (!newKey || newKey === oldKey || newKey in record.value) return;
  const { [oldKey]: val, ...rest } = record.value as Record<string, unknown>;
  emit('update:modelValue', { ...rest, [newKey]: val });
}

function removeKey(key: string) {
  const { [key]: _omit, ...rest } = record.value as Record<string, unknown>;
  emit('update:modelValue', rest);
}

function addEntry() {
  const key = pendingKey.value;
  if (!key || key in record.value) return;
  emit('update:modelValue', { ...record.value, [key]: makeDefault(valueSchema.value) });
  pendingKey.value = '';
}

function setJSONEntry(key: string, raw: string) {
  try {
    setEntry(key, JSON.parse(raw));
  } catch {
    // ignore until the raw value parses as JSON
  }
}
</script>

<template>
  <div class="flex flex-col gap-2">
    <div
      v-for="key in entries"
      :key="key"
      class="flex items-start gap-2 rounded-lg border border-slate-200 p-2 dark:border-white/10"
    >
      <input
        :value="key"
        @input="renameKey(key, ($event.target as HTMLInputElement).value)"
        title="Map key"
        class="rounded border border-slate-300 bg-white font-mono text-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
      />
      <div class="flex-1">
        <SchemaField
          v-if="valueSchema"
          :schema="valueSchema"
          :model-value="record[key]"
          @update:model-value="setEntry(key, $event)"
        />
        <textarea
          v-else
          :value="JSON.stringify(record[key], null, 2)"
          @input="setJSONEntry(key, ($event.target as HTMLTextAreaElement).value)"
          placeholder="JSON value"
          class="rounded border border-slate-300 bg-white font-mono text-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
        />
        <p v-if="props.schema?.description" class="mt-1 text-[10px] text-slate-500">
          {{ props.schema.description }}
        </p>
      </div>
      <button
        type="button"
        title="Remove entry"
        @click="removeKey(key)"
        class="text-[10px] text-slate-400 hover:text-rose-600"
      >
        ✕
      </button>
    </div>

    <div class="flex items-end gap-2 pt-1">
      <label class="text-xs text-slate-500 dark:text-slate-400">Add entry</label>
      <input
        v-model="pendingKey"
        placeholder="key"
        class="rounded border border-slate-300 bg-white font-mono text-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
      />
      <button
        type="button"
        :disabled="!pendingKey"
        @click="addEntry"
        class="rounded bg-violet-600 px-2 py-1 text-xs font-bold text-white hover:bg-violet-500 disabled:opacity-40"
      >
        Add
      </button>
    </div>

    <p v-if="!entries.length" class="text-[10px] text-slate-500">No entries — add one below.</p>
  </div>
</template>
