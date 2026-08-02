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
      :class="triggerClasses"
      @click="toggle"
      @keydown.down.prevent="handleTriggerDown"
      @keydown.up.prevent="handleTriggerUp"
      @keydown.escape="close"
    >
      <slot name="trigger" :label="displayLabel" :is-open="isOpen" :selected-item="selectedItem">
        <span class="block min-w-0 flex-1 overflow-hidden text-ellipsis whitespace-nowrap">
          {{ displayLabel }}
        </span>
      </slot>
      <svg
        class="ml-1.5 h-3.5 w-3.5 shrink-0 text-slate-400 transition-transform duration-200"
        :class="{ 'rotate-180': isOpen }"
        fill="none" stroke="currentColor" viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
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
        class="absolute z-[60] mt-1 w-max min-w-[220px] max-w-[min(32rem,80vw)] overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg shadow-slate-900/5 dark:border-white/10 dark:bg-slate-900 dark:shadow-slate-950/30"
      >
        <!-- Search input -->
        <div v-if="searchable" class="border-b border-slate-100 p-1.5 dark:border-white/5">
          <div class="relative">
            <svg
              class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400"
              fill="none" stroke="currentColor" viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              :placeholder="searchPlaceholder"
              class="h-7 w-full rounded-md border-0 bg-slate-50 pl-8 pr-2.5 text-[0.8rem] outline-none transition-colors placeholder:text-slate-400 focus:bg-slate-100 dark:bg-slate-800 dark:text-slate-200 dark:placeholder:text-slate-500 dark:focus:bg-slate-800/80"
              @keydown.down.prevent="highlightNext"
              @keydown.up.prevent="highlightPrev"
              @keydown.enter.prevent="selectHighlighted"
              @keydown.escape.prevent="close"
              @click.stop
            />
          </div>
        </div>

        <!-- Items list -->
        <ul ref="listRef" class="max-h-56 overflow-y-auto p-1" role="listbox">
          <li
            v-for="(item, index) in filteredItems"
            :key="item.value"
            role="option"
            :aria-selected="item.value === modelValue"
            class="flex min-w-0 cursor-pointer items-center gap-2 rounded-md px-2.5 py-1.5 text-[0.8rem] transition-colors duration-75"
            :class="getItemClasses(item, index)"
            @click="select(item)"
            @mouseenter="highlightedIndex = index"
          >
            <slot
              name="item"
              :item="item"
              :is-selected="item.value === modelValue"
              :is-highlighted="highlightedIndex === index"
            >
              <span class="min-w-0 flex-1 whitespace-normal break-words">{{ item.label }}</span>
            </slot>
            <svg
              v-if="item.value === modelValue"
              class="h-3.5 w-3.5 shrink-0 text-indigo-500 dark:text-indigo-400"
              fill="none" stroke="currentColor" viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
            </svg>
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

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'

export interface SelectItem {
  value: string
  label: string
  disabled?: boolean
  [key: string]: any
}

const props = withDefaults(defineProps<{
  items: SelectItem[]
  modelValue: string
  placeholder?: string
  disabled?: boolean
  searchable?: boolean
  searchPlaceholder?: string
  title?: string
}>(), {
  placeholder: 'Select...',
  disabled: false,
  searchable: false,
  searchPlaceholder: 'Search...',
  title: undefined,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const triggerRef = ref<HTMLButtonElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLUListElement | null>(null)
const isOpen = ref(false)
const searchQuery = ref('')
const highlightedIndex = ref(-1)

const selectedItem = computed(() =>
  props.items.find(item => item.value === props.modelValue) ?? null
)

const displayLabel = computed(() =>
  selectedItem.value?.label ?? props.placeholder
)

const filteredItems = computed(() => {
  if (!props.searchable || !searchQuery.value) return props.items
  const query = searchQuery.value.toLowerCase()
  return props.items.filter(item =>
    item.label.toLowerCase().includes(query) ||
    item.value.toLowerCase().includes(query)
  )
})

const triggerClasses = computed(() => [
  'flex h-8 w-full items-center justify-between rounded-lg border px-2.5 text-left text-[0.8rem] font-medium transition-all duration-150 outline-none',
  'border-slate-200 bg-white text-slate-700',
  'hover:border-slate-300',
  'focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20',
  'disabled:cursor-not-allowed disabled:opacity-50',
  'dark:border-white/10 dark:bg-slate-900 dark:text-slate-200 dark:hover:border-white/20',
  isOpen.value && 'border-indigo-400 ring-2 ring-indigo-500/20',
])

function getItemClasses(item: SelectItem, index: number): Record<string, boolean> {
  const isSelected = item.value === props.modelValue
  const isHighlighted = highlightedIndex.value === index
  return {
    'bg-indigo-50 text-indigo-700 dark:bg-indigo-500/10 dark:text-indigo-300':
      isSelected && !isHighlighted,
    'bg-indigo-100 text-indigo-800 dark:bg-indigo-500/15 dark:text-indigo-200':
      isSelected && isHighlighted,
    'bg-slate-100 text-slate-900 dark:bg-white/5 dark:text-slate-100':
      !isSelected && isHighlighted,
    'text-slate-700 dark:text-slate-300':
      !isSelected && !isHighlighted,
    'opacity-50 pointer-events-none': item.disabled === true,
  }
}

function toggle() {
  isOpen.value ? close() : open()
}

function open() {
  if (props.disabled) return
  isOpen.value = true
  searchQuery.value = ''
  highlightedIndex.value = -1
}

function close() {
  isOpen.value = false
  searchQuery.value = ''
  highlightedIndex.value = -1
  triggerRef.value?.focus()
}

function select(item: SelectItem) {
  if (item.disabled) return
  emit('update:modelValue', item.value)
  close()
}

function selectHighlighted() {
  if (highlightedIndex.value >= 0 && highlightedIndex.value < filteredItems.value.length) {
    select(filteredItems.value[highlightedIndex.value])
  }
}

function highlightNext() {
  if (filteredItems.value.length === 0) return
  highlightedIndex.value = (highlightedIndex.value + 1) % filteredItems.value.length
  scrollToHighlighted()
}

function highlightPrev() {
  if (filteredItems.value.length === 0) return
  highlightedIndex.value =
    highlightedIndex.value <= 0
      ? filteredItems.value.length - 1
      : highlightedIndex.value - 1
  scrollToHighlighted()
}

function scrollToHighlighted() {
  nextTick(() => {
    if (!listRef.value || highlightedIndex.value < 0) return
    const items = listRef.value.querySelectorAll('li')
    items[highlightedIndex.value]?.scrollIntoView({ block: 'nearest' })
  })
}

function handleTriggerDown() {
  if (!isOpen.value) open()
  nextTick(() => highlightNext())
}

function handleTriggerUp() {
  if (!isOpen.value) open()
  nextTick(() => highlightPrev())
}

watch(isOpen, async (val) => {
  if (val) {
    await nextTick()
    if (props.searchable) {
      searchInputRef.value?.focus()
    }
  }
})

watch(searchQuery, () => {
  highlightedIndex.value = -1
})
</script>