<template>
  <Transition name="dropdown">
    <div
      class="absolute inset-x-0 bottom-full z-50 mb-2 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-2xl shadow-slate-300/20 dark:border-white/10 dark:bg-slate-900 dark:shadow-black/30"
      @mousedown.prevent
    >
      <!-- ── Header ── -->
      <div class="flex items-center justify-between border-b border-slate-100 px-3 py-2 dark:border-white/5">
        <div class="flex items-center gap-2">
          <span class="text-xs text-violet-500">🔍</span>
          <span v-if="loading" class="text-xs font-medium text-slate-400">Searching…</span>
          <span v-else class="text-xs font-medium text-slate-400 dark:text-slate-500">
            {{ results.length }} result{{ results.length === 1 ? '' : 's' }}
          </span>
        </div>
        <div class="hidden items-center gap-1.5 text-[0.65rem] text-slate-400 sm:flex dark:text-slate-500">
          <kbd class="rounded border border-slate-200 px-1 dark:border-white/10">↑↓</kbd>
          <span>navigate</span>
          <kbd class="rounded border border-slate-200 px-1 dark:border-white/10">↵</kbd>
          <span>select</span>
          <kbd class="rounded border border-slate-200 px-1 dark:border-white/10">esc</kbd>
          <span>cancel</span>
        </div>
      </div>

      <!-- ── Scrollable list ── -->
      <div ref="listRef" class="max-h-72 overflow-y-auto overscroll-contain p-1.5">
        <!-- Loading spinner -->
        <div v-if="loading" class="flex items-center justify-center py-8">
          <div class="h-5 w-5 animate-spin rounded-full border-2 border-violet-500 border-t-transparent"></div>
        </div>

        <!-- No query yet -->
        <div v-else-if="!query.trim()" class="flex flex-col items-center gap-1 py-8 text-center">
          <span class="text-2xl">💡</span>
          <p class="text-sm text-slate-400 dark:text-slate-500">Type to search your prompts</p>
        </div>

        <!-- Grouped results -->
        <template v-else-if="grouped.length">
          <template v-for="group in grouped" :key="group.key">
            <!-- Tag header -->
            <div v-if="group.name" class="flex items-center gap-1.5 px-2 pt-2.5 pb-1">
              <span class="text-[0.65rem]">🏷️</span>
              <span class="text-[0.7em] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">
                {{ group.name }}
              </span>
            </div>

            <!-- Prompt items -->
            <button
              v-for="(prompt, idx) in group.items"
              :key="prompt.id"
              :ref="(el: any) => setItemRef(globalIndex(group, idx), el)"
              type="button"
              class="flex w-full items-start gap-3 rounded-lg px-2.5 py-2.5 text-left transition-colors"
              :class="[
                activeIndex === globalIndex(group, idx)
                  ? 'bg-violet-50 dark:bg-violet-500/10'
                  : 'hover:bg-slate-50 dark:hover:bg-white/5',
              ]"
              @click="choose(prompt)"
            >
              <!-- Left accent bar for active item -->
              <div
                class="mt-0.5 h-8 w-0.5 shrink-0 rounded-full transition-colors"
                :class="activeIndex === globalIndex(group, idx) ? 'bg-violet-500' : 'bg-transparent'"
              ></div>

              <!-- Content -->
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="truncate text-sm font-semibold text-slate-900 dark:text-white">
                    {{ prompt.title || 'Untitled' }}
                  </span>
                  <span
                    v-if="(prompt.tags || []).length"
                    class="shrink-0 rounded-full bg-slate-100 px-1.5 py-0.5 text-[0.6em] font-medium text-slate-500 dark:bg-white/10 dark:text-slate-400"
                  >
                    {{ prompt.tags![0] }}
                  </span>
                </div>
                <p class="mt-0.5 line-clamp-1 text-xs text-slate-500 dark:text-slate-400">
                  {{ prompt.description || prompt.content }}
                </p>
              </div>

              <!-- Enter hint for active item -->
              <span
                v-if="activeIndex === globalIndex(group, idx)"
                class="mt-1 shrink-0 rounded bg-violet-100 px-1.5 py-0.5 text-[0.6em] font-bold text-violet-600 dark:bg-violet-500/20 dark:text-violet-300"
              >
                ↵
              </span>
            </button>
          </template>
        </template>

        <!-- No results -->
        <div v-else class="flex flex-col items-center gap-1 py-8 text-center">
          <span class="text-2xl">🔍</span>
          <p class="text-sm text-slate-400 dark:text-slate-500">No matching prompts</p>
          <p class="text-xs text-slate-300 dark:text-slate-600">Try a different search term</p>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import type { PromptResponse } from '../../types'
import { computed, nextTick, ref, watch, type PropType } from 'vue'

const props = defineProps({
  results: { type: Array as PropType<PromptResponse[]>, default: () => [] },
  loading: { type: Boolean, default: false },
  query: { type: String, default: '' },
})

const emit = defineEmits<{
  select: [prompt: PromptResponse]
}>()

/* ───── active index & keyboard nav ───── */
const activeIndex = ref(0)
const listRef = ref<HTMLElement | null>(null)
const itemEls = ref<Record<number, HTMLElement>>({})

function setItemRef(index: number, el: any) {
  if (el) itemEls.value[index] = el as HTMLElement
}

/* Reset when results change */
watch(() => props.results.length, () => {
  activeIndex.value = 0
})

/* Scroll active item into view */
watch(activeIndex, (idx) => {
  nextTick(() => {
    itemEls.value[idx]?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
})

/* ───── grouping by first tag ───── */
const grouped = computed(() => {
  const map = new Map<string, PromptResponse[]>()
  for (const p of props.results) {
    const key = p.tags?.[0] || '__untagged__'
    const list = map.get(key) || []
    list.push(p)
    map.set(key, list)
  }
  return Array.from(map.entries()).map(([tagName, items]) => ({
    key: tagName,
    name: tagName === '__untagged__' ? '' : tagName,
    items,
  }))
})

function globalIndex(group: { items: PromptResponse[] }, idx: number): number {
  let acc = 0
  for (const g of grouped.value) {
    if (g === group) return acc + idx
    acc += g.items.length
  }
  return acc
}

const flatLength = computed(() => props.results.length)

/* ───── actions ───── */
function choose(prompt: PromptResponse) {
  emit('select', prompt)
}

function move(delta: number) {
  if (!flatLength.value) return
  activeIndex.value = (activeIndex.value + delta + flatLength.value) % flatLength.value
}

function activate() {
  const prompt = props.results[activeIndex.value]
  if (prompt) choose(prompt)
}

defineExpose({ move, activate })
</script>

<style scoped>
.line-clamp-1 {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-clamp: 1;
  -webkit-line-clamp: 1;
}

.dropdown-enter-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.dropdown-leave-active {
  transition: opacity 0.1s ease, transform 0.1s ease;
}
.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(6px);
}
</style>