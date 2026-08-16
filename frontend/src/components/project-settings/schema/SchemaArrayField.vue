<script setup lang="ts">
import { computed, ref } from 'vue';
import type { SchemaNode, SchemaValue } from './schemaUtils';
import {
  itemSchemaOf,
  isEnumSchema,
  isObjectSchema,
  isNumberSchema,
  isBooleanSchema,
  makeDefault,
  makeDefaultObject,
} from './schemaUtils';
import SchemaField from './SchemaField.vue';

const props = defineProps<{
  schema?: SchemaNode;
  modelValue?: SchemaValue;
}>();
const emit = defineEmits<{ (e: 'update:modelValue', value: unknown[]): void }>();

const itemSchema = computed(() => itemSchemaOf(props.schema));
const items = computed(() => {
  const v = props.modelValue;
  return Array.isArray(v) ? (v as unknown[]) : [];
});

const isObjectItem = computed(() => !!itemSchema.value && isObjectSchema(itemSchema.value));
const isEnumItem = computed(() => !!itemSchema.value && isEnumSchema(itemSchema.value));
const isScalarItem = computed(() => !!itemSchema.value && !isObjectItem.value && !isEnumItem.value);

const pendingScalar = ref('');

function updateItem(index: number, value: unknown) {
  const next = items.value.slice();
  next[index] = value;
  emit('update:modelValue', next);
}

function removeAt(index: number) {
  const next = items.value.slice();
  next.splice(index, 1);
  emit('update:modelValue', next);
}

function move(index: number, delta: number) {
  const next = items.value.slice();
  const target = next[index + delta];
  if (target === undefined) return;
  [next[index], next[index + delta]] = [target, next[index]];
  emit('update:modelValue', next);
}

function defaultItem(): unknown {
  if (isObjectItem.value) return makeDefaultObject(itemSchema.value!);
  if (itemSchema.value) return makeDefault(itemSchema.value);
  return '';
}

function add() {
  emit('update:modelValue', [...items.value, defaultItem()]);
}

function parseScalar(raw: string): unknown {
  if (raw.trim() === '') return undefined;
  if (isNumberSchema(itemSchema.value!)) {
    const n = Number(raw);
    return Number.isNaN(n) ? undefined : n;
  }
  if (isBooleanSchema(itemSchema.value!)) {
    if (raw === 'true') return true;
    if (raw === 'false') return false;
    return undefined;
  }
  return String(raw);
}

function addScalar() {
  const parsed = parseScalar(pendingScalar.value);
  if (parsed === undefined) return;
  emit('update:modelValue', [...items.value, parsed]);
  pendingScalar.value = '';
}

function toggleEnum(option: unknown, checked: boolean) {
  const next = items.value.slice();
  if (checked && !next.includes(option)) next.push(option);
  if (!checked) {
    const idx = next.findIndex((x) => String(x) === String(option));
    if (idx >= 0) next.splice(idx, 1);
  }
  emit('update:modelValue', next);
}

function inArray(option: unknown): boolean {
  return items.value.some((x) => String(x) === String(option));
}
</script>

<template>
  <div class="flex flex-col gap-2">
    <div v-if="isObjectItem" class="flex flex-col gap-2">
      <div
        v-for="(item, index) in items"
        :key="index"
        class="rounded-lg border border-slate-200 p-3 dark:border-white/10"
      >
        <div class="mb-1 flex items-center justify-between">
          <span class="text-xs font-medium text-slate-500">item {{ index + 1 }}</span>
          <div class="flex items-center gap-1">
            <button
              type="button"
              title="Move up"
              @click="move(index, -1)"
              class="text-[10px] text-slate-400 hover:text-slate-700"
            >
              ▲
            </button>
            <button
              type="button"
              title="Move down"
              @click="move(index, 1)"
              class="text-[10px] text-slate-400 hover:text-slate-700"
            >
              ▼
            </button>
            <button
              type="button"
              title="Remove"
              @click="removeAt(index)"
              class="text-[10px] text-slate-400 hover:text-rose-600"
            >
              ✕
            </button>
          </div>
        </div>
        <SchemaField
          :schema="itemSchema"
          :model-value="item"
          @update:model-value="updateItem(index, $event)"
        />
      </div>
      <button
        type="button"
        @click="add"
        class="text-xs font-bold text-violet-600 hover:text-violet-800"
      >
        + Add item
      </button>
    </div>
    <div v-else-if="isEnumItem" class="flex flex-wrap gap-2">
      <label
        v-for="option in itemSchema!.enum"
        :key="String(option)"
        class="flex items-center gap-1.5 text-sm"
      >
        <input
          type="checkbox"
          :value="String(option)"
          :checked="inArray(option)"
          @change="toggleEnum(option, ($event.target as HTMLInputElement).checked)"
        />
        <span>{{ String(option) }}</span>
      </label>
    </div>

    <div v-else-if="isScalarItem" class="flex flex-wrap items-center gap-1.5">
      <div
        v-for="(item, index) in items"
        :key="index"
        class="flex items-center gap-1 rounded border border-slate-300 px-2 py-1 dark:border-white/10"
      >
        <SchemaField
          :name="String(index)"
          :schema="itemSchema"
          :model-value="item"
          @update:model-value="updateItem(index, $event)"
        />
        <button
          type="button"
          title="Remove"
          @click="removeAt(index)"
          class="text-[10px] text-slate-400 hover:text-rose-600"
        >
          ✕
        </button>
      </div>
      <input
        v-model="pendingScalar"
        @keydown.enter="addScalar"
        placeholder="add value and press ⏎"
        class="rounded border border-slate-300 bg-white text-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
      />
      <p v-if="!items.length" class="text-[10px] text-slate-500">No items yet.</p>
    </div>

    <div v-else class="text-xs text-slate-500">
      Array editor unavailable for this item type (use Code mode).
    </div>
  </div>
</template>
