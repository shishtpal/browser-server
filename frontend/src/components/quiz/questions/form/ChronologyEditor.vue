<template>
  <div
    class="space-y-4 rounded-xl border border-gray-200 bg-gray-50/50 p-3 sm:rounded-2xl sm:p-5 dark:border-slate-700 dark:bg-slate-800/30"
  >
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="min-w-0">
        <span class="block text-sm font-bold text-slate-700 dark:text-slate-300">Items</span>
        <span class="block text-xs text-slate-500 dark:text-slate-400">
          Set the correct order number for each item
        </span>
      </div>
      <Button
        variant="secondary"
        size="sm"
        class="shrink-0"
        :disabled="items.length >= maxItems"
        @click="addItem"
      >
        <span class="inline-flex items-center gap-1">
          <Plus class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Add item
        </span>
      </Button>
    </div>

    <TransitionGroup tag="div" name="row" class="space-y-2">
      <div v-for="(item, i) in items" :key="i" class="flex items-center gap-2 sm:gap-3">
        <span
          class="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-violet-100 text-xs font-black text-violet-700 dark:bg-violet-900/30 dark:text-violet-300"
          aria-hidden="true"
        >
          #{{ i + 1 }}
        </span>
        <InputField
          v-model.number="item.correct_order"
          class="w-16 shrink-0 sm:w-20"
          type="number"
          aria-label="Correct order"
        />
        <InputField
          v-model="item.text"
          class="min-w-0 flex-1"
          :placeholder="`Item ${i + 1}`"
          flex
        />
        <button
          type="button"
          class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-slate-400 transition hover:bg-rose-50 hover:text-rose-500 disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-slate-400 dark:hover:bg-rose-900/20"
          :disabled="items.length <= minItems"
          aria-label="Remove item"
          @click="removeItem(i)"
        >
          <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { Plus, Trash2 } from '@lucide/vue';
import Button from '../../../ui/Button.vue';
import InputField from '../../../ui/InputField.vue';

export interface ChronologyDraft {
  index: number;
  text: string;
  correct_order: number;
}

const props = withDefaults(
  defineProps<{
    items: ChronologyDraft[];
    minItems?: number;
    maxItems?: number;
  }>(),
  {
    minItems: 2,
    maxItems: 20,
  },
);

const addItem = () => {
  if (props.items.length >= props.maxItems) return;
  props.items.push({
    index: props.items.length,
    text: '',
    correct_order: props.items.length + 1,
  });
};

const removeItem = (index: number) => {
  props.items.splice(index, 1);
  props.items.forEach((c, idx) => (c.index = idx));
};
</script>

<style scoped>
.row-enter-active,
.row-leave-active {
  transition: all 0.2s ease;
}
.row-enter-from,
.row-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
