<script setup lang="ts">
import { computed } from 'vue';
import type { SchemaNode } from './schemaUtils';
import {
  fieldIsArray,
  fieldIsMap,
  fieldIsObject,
  humanizeKey,
  isBooleanSchema,
  isEnumSchema,
  isNumberSchema,
  isSecretName,
  type SchemaValue,
} from './schemaUtils';
import SchemaArrayField from './SchemaArrayField.vue';
import SchemaForm from './SchemaForm.vue';
import SchemaMapField from './SchemaMapField.vue';

const props = withDefaults(
  defineProps<{
    name?: string;
    schema?: SchemaNode;
    modelValue?: SchemaValue;
    required?: boolean;
  }>(),
  { name: 'field', required: false },
);
const emit = defineEmits<{
  (e: 'update:modelValue', value: SchemaValue): void;
}>();

const isObject = computed(() => fieldIsObject(props.schema));
const isMap = computed(() => fieldIsMap(props.schema));
const isArray = computed(() => fieldIsArray(props.schema));
const isEnum = computed(() => isEnumSchema(props.schema));
const isToggle = computed(() => isBooleanSchema(props.schema));
const isNumber = computed(() => isNumberSchema(props.schema));
const isSecret = computed(() => isSecretName(props.name ?? ''));
const enumOptions = computed(() => (props.schema?.enum ? [...props.schema.enum] : []));

const label = computed(() => humanizeKey(props.name ?? 'field'));

function emitValue(value: unknown) {
  emit('update:modelValue', value);
}

function onNumber(value: string) {
  if (value === '') {
    emitValue(undefined);
    return;
  }
  const parsed = Number(value);
  emitValue(Number.isNaN(parsed) ? value : parsed);
}

function onToggle() {
  emitValue(!props.modelValue);
}

function onString(value: string) {
  emitValue(value === '' ? undefined : value);
}

function onEnum(value: string) {
  if (value === '' && !props.required) {
    emitValue(undefined);
    return;
  }
  const match = enumOptions.value.find((o) => String(o) === value);
  emitValue(match === undefined ? value : match);
}
</script>

<template>
  <SchemaForm
    v-if="isObject"
    :schema="props.schema"
    :model-value="modelValue as Record<string, unknown> | null"
    @update:model-value="emitValue"
  />
  <SchemaMapField
    v-else-if="isMap"
    :schema="props.schema"
    :model-value="modelValue"
    @update:model-value="emitValue"
  />
  <SchemaArrayField
    v-else-if="isArray"
    :schema="props.schema"
    :model-value="modelValue"
    @update:model-value="emitValue"
  />
  <div v-else class="flex flex-col gap-1.5">
    <label class="flex flex-col gap-1">
      <span class="block text-xs font-medium text-slate-700 dark:text-slate-200">
        {{ label }}
        <span
          v-if="props.required"
          class="text-rose-600"
          title="Required by schema"
          aria-hidden="true"
          >*</span
        >
      </span>
      <p
        v-if="props.schema?.description"
        class="text-[10px] leading-relaxed text-slate-500 dark:text-slate-400"
      >
        {{ props.schema.description }}
      </p>

      <select
        v-if="isEnum"
        :value="modelValue === undefined || modelValue === null ? '' : String(modelValue)"
        @change="onEnum(($event.target as HTMLSelectElement).value)"
        class="rounded border border-slate-300 bg-white text-sm focus:ring-1 focus:ring-violet-500/70 focus:outline-none dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
      >
        <option v-if="!required" value="">— optional —</option>
        <option v-for="option in enumOptions" :key="String(option)" :value="String(option)">
          {{ String(option) }}
        </option>
      </select>

      <button
        v-else-if="isToggle"
        type="button"
        role="switch"
        :aria-checked="!!modelValue"
        @click="onToggle"
        :class="modelValue ? 'bg-violet-600' : 'bg-slate-300 dark:bg-slate-600'"
        class="relative inline-flex h-5 w-9 items-center rounded-full transition outline-none"
      >
        <span
          :class="modelValue ? 'translate-x-5' : 'translate-x-1'"
          class="inline-block h-4 w-4 transform rounded-full bg-white transition"
        />
      </button>

      <input
        v-else-if="isNumber"
        type="number"
        :min="props.schema?.minimum"
        :max="props.schema?.maximum"
        :value="modelValue === undefined || modelValue === null ? '' : String(modelValue)"
        @input="onNumber(($event.target as HTMLInputElement).value)"
        class="rounded border border-slate-300 bg-white text-sm focus:ring-1 focus:ring-violet-500/70 focus:outline-none dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
      />

      <input
        v-else
        :type="isSecret ? 'password' : 'text'"
        :value="modelValue === undefined || modelValue === null ? '' : String(modelValue)"
        @input="onString(($event.target as HTMLInputElement).value)"
        class="rounded border border-slate-300 bg-white text-sm focus:ring-1 focus:ring-violet-500/70 focus:outline-none dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
      />
    </label>
  </div>
</template>
