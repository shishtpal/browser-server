<template>
  <div ref="containerRef" class="relative">
    <!-- Trigger button -->
    <button
      ref="triggerRef"
      type="button"
      :disabled="disabled"
      :title="title"
      :aria-expanded="isOpen"
      aria-haspopup="listbox"
      aria-multiselectable="true"
      :class="triggerClasses"
      @click="toggle"
      @keydown.down.prevent="handleTriggerDown"
      @keydown.up.prevent="handleTriggerUp"
      @keydown.escape="close"
    >
      <slot name="trigger" :label="displayLabel" :is-open="isOpen" :selected-items="selectedItems">
        <span class="block min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap">
          {{ displayLabel }}
        </span>
      </slot>
      <svg
        class="ml-1.5 h-3.5 w-3.5 shrink-0 text-slate-400 transition-transform duration-200"
        :class="{ 'rotate-180': isOpen }"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
      </svg>
    </button>

    <!-- Dropdown panel -->
    <transition
      enter-active-class="transition ease-out duration-150"
      enter-from-class="opacity-0 -translate-y-1 scale-95"
      enter-to-class="opacity-100 translate-y-0 scale-100"
      leave-active-class="transition ease-in duration-100"
      leave-from-class="opacity-100 translate-y-0 scale-100"
      leave-to-class="opacity-0 -translate-y-1 scale-95"
    >
      <div
        v-if="isOpen"
        class="absolute z-[60] mt-1 w-max max-w-[min(32rem,80vw)] min-w-[220px] overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg shadow-slate-900/5 dark:border-white/10 dark:bg-slate-900 dark:shadow-slate-950/30"
        :class="align === 'right' ? 'right-0' : 'left-0'"
      >
        <!-- Search input -->
        <div v-if="searchable" class="border-b border-slate-100 p-1.5 dark:border-white/5">
          <div class="relative">
            <svg
              class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              :placeholder="searchPlaceholder"
              class="h-7 w-full rounded-md border-0 bg-slate-50 pr-2.5 pl-8 text-[0.8rem] transition-colors outline-none placeholder:text-slate-400 focus:bg-slate-100 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:bg-slate-800/80"
              @keydown.down.prevent="highlightNext"
              @keydown.up.prevent="highlightPrev"
              @keydown.enter.prevent="selectHighlighted"
              @keydown.escape.prevent="close"
              @click.stop
            />
          </div>
        </div>

        <!-- Items list -->
        <ul
          ref="listRef"
          class="max-h-56 overflow-y-auto p-1"
          role="listbox"
          aria-multiselectable="true"
        >
          <li
            v-for="(item, index) in filteredItems"
            :key="item.value"
            role="option"
            :aria-selected="isSelected(item.value)"
            class="flex min-w-0 cursor-pointer items-center gap-2 rounded-md px-2.5 py-1.5 text-[0.8rem] transition-colors duration-75"
            :class="getItemClasses(item, index)"
            @click="select(item)"
            @mouseenter="highlightedIndex = index"
          >
            <slot
              name="item"
              :item="item"
              :is-selected="isSelected(item.value)"
              :is-highlighted="highlightedIndex === index"
            >
              <!-- Checkbox indicator -->
              <span
                class="grid h-3.5 w-3.5 shrink-0 place-items-center rounded border transition-colors"
                :class="
                  isSelected(item.value)
                    ? 'border-indigo-500 bg-indigo-500 text-white'
                    : 'border-slate-300 bg-white dark:border-white/20 dark:bg-slate-800'
                "
                aria-hidden="true"
              >
                <svg
                  v-if="isSelected(item.value)"
                  class="h-2.5 w-2.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="3"
                    d="M5 13l4 4L19 7"
                  />
                </svg>
              </span>
              <span class="min-w-0 flex-1 break-words whitespace-normal">{{ item.label }}</span>
            </slot>
          </li>
        </ul>

        <!-- Empty state -->
        <div
          v-if="filteredItems.length === 0"
          class="px-3 py-4 text-center text-[0.8rem] text-slate-400"
        >
          <slot name="empty">No results found</slot>
        </div>
      </div>
    </transition>

    <!-- Click-outside overlay -->
    <div v-if="isOpen" class="fixed inset-0 z-40" @click="close" />
  </div>
</template>

<script lang="ts">
import type { SelectItem } from './SearchableSelect.vue';
export type { SelectItem };
</script>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue';

const props = withDefaults(
  defineProps<{
    items: SelectItem[];
    modelValue: string[];
    placeholder?: string;
    disabled?: boolean;
    searchable?: boolean;
    searchPlaceholder?: string;
    title?: string;
    /** Panel horizontal alignment relative to the trigger. */
    align?: 'left' | 'right';
  }>(),
  {
    placeholder: 'Select...',
    disabled: false,
    searchable: false,
    searchPlaceholder: 'Search...',
    title: undefined,
    align: 'left',
  },
);

const emit = defineEmits<{
  'update:modelValue': [value: string[]];
}>();

const triggerRef = ref<HTMLButtonElement | null>(null);
const searchInputRef = ref<HTMLInputElement | null>(null);
const listRef = ref<HTMLUListElement | null>(null);
const isOpen = ref(false);
const searchQuery = ref('');
const highlightedIndex = ref(-1);

const isSelected = (value: string) => props.modelValue.includes(value);

const selectedItems = computed(() => props.items.filter((item) => isSelected(item.value)));

const displayLabel = computed(() => {
  if (selectedItems.value.length === 0) return props.placeholder;
  return selectedItems.value.map((item) => item.label).join(', ');
});

const filteredItems = computed(() => {
  if (!props.searchable || !searchQuery.value) return props.items;
  const query = searchQuery.value.toLowerCase();
  return props.items.filter(
    (item) => item.label.toLowerCase().includes(query) || item.value.toLowerCase().includes(query),
  );
});

const triggerClasses = computed(() => [
  'flex h-8 w-full items-center justify-between rounded-lg border px-2.5 text-left text-[0.8rem] font-medium transition-all duration-150 outline-none',
  'border-slate-200 bg-white text-slate-700',
  'hover:border-slate-300',
  'focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20',
  'disabled:cursor-not-allowed disabled:opacity-50',
  'dark:border-white/10 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-white/20',
  isOpen.value && 'border-indigo-400 ring-2 ring-indigo-500/20',
]);

function getItemClasses(item: SelectItem, index: number): Record<string, boolean> {
  const selected = isSelected(item.value);
  const isHighlighted = highlightedIndex.value === index;
  return {
    'bg-indigo-50 text-indigo-700 dark:bg-indigo-500/10 dark:text-indigo-300':
      selected && !isHighlighted,
    'bg-indigo-100 text-indigo-800 dark:bg-indigo-500/15 dark:text-indigo-200':
      selected && isHighlighted,
    'bg-slate-100 text-slate-900 dark:bg-white/5 dark:text-slate-100': !selected && isHighlighted,
    'text-slate-700 dark:text-slate-300': !selected && !isHighlighted,
    'opacity-50 pointer-events-none': item.disabled === true,
  };
}

function toggle() {
  isOpen.value ? close() : open();
}

function open() {
  if (props.disabled) return;
  isOpen.value = true;
  searchQuery.value = '';
  highlightedIndex.value = -1;
}

function close() {
  isOpen.value = false;
  searchQuery.value = '';
  highlightedIndex.value = -1;
  triggerRef.value?.focus();
}

function select(item: SelectItem) {
  if (item.disabled) return;
  const next = isSelected(item.value)
    ? props.modelValue.filter((v) => v !== item.value)
    : [...props.modelValue, item.value];
  emit('update:modelValue', next);
  // Dropdown intentionally stays open for multi-select.
}

function selectHighlighted() {
  if (highlightedIndex.value >= 0 && highlightedIndex.value < filteredItems.value.length) {
    select(filteredItems.value[highlightedIndex.value]);
  }
}

function highlightNext() {
  if (filteredItems.value.length === 0) return;
  highlightedIndex.value = (highlightedIndex.value + 1) % filteredItems.value.length;
  scrollToHighlighted();
}

function highlightPrev() {
  if (filteredItems.value.length === 0) return;
  highlightedIndex.value =
    highlightedIndex.value <= 0 ? filteredItems.value.length - 1 : highlightedIndex.value - 1;
  scrollToHighlighted();
}

function scrollToHighlighted() {
  nextTick(() => {
    if (!listRef.value || highlightedIndex.value < 0) return;
    const items = listRef.value.querySelectorAll('li');
    items[highlightedIndex.value]?.scrollIntoView({ block: 'nearest' });
  });
}

function handleTriggerDown() {
  if (!isOpen.value) open();
  nextTick(() => highlightNext());
}

function handleTriggerUp() {
  if (!isOpen.value) open();
  nextTick(() => highlightPrev());
}

watch(isOpen, async (val) => {
  if (val) {
    await nextTick();
    if (props.searchable) {
      searchInputRef.value?.focus();
    }
  }
});

watch(searchQuery, () => {
  highlightedIndex.value = -1;
});
</script>
