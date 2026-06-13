<script setup lang="ts">
import { computed, ref, useTemplateRef, watch } from 'vue'
import HistoryPanel from './HistoryPanel.vue'
import TodosPanel from './TodosPanel.vue'

const props = defineProps<{ initialPanel: 'history' | 'todos' }>()

const activePanel = ref<'history' | 'todos'>(props.initialPanel)
const stats = ref('Loading…')

const historyPanel = useTemplateRef<InstanceType<typeof HistoryPanel>>('historyPanel')
const todosPanel = useTemplateRef<InstanceType<typeof TodosPanel>>('todosPanel')

function updateStats(label: string) {
  stats.value = label
}

const tabHistoryClass = computed(() =>
  activePanel.value === 'history'
    ? 'border-b-2 border-rose-400 px-3 py-2 text-sm font-medium text-rose-300'
    : 'border-b-2 border-transparent px-3 py-2 text-sm font-medium text-slate-400',
)

const tabTodosClass = computed(() =>
  activePanel.value === 'history'
    ? 'border-b-2 border-transparent px-3 py-2 text-sm font-medium text-slate-400'
    : 'border-b-2 border-rose-400 px-3 py-2 text-sm font-medium text-rose-300',
)

function refreshHistory() {
  historyPanel.value?.refresh()
}

function clearHistory() {
  historyPanel.value?.clearAll()
}

function refreshTodos() {
  todosPanel.value?.refresh()
}

function clearTodos() {
  todosPanel.value?.clearAll()
}

watch(activePanel, () => {
  if (activePanel.value === 'history') {
    updateStats('Loading…')
  } else {
    updateStats('0 todos · 0 done')
  }
})
</script>

<template>
  <main class="w-[480px] max-h-[540px] overflow-hidden rounded-xl border border-slate-800 bg-slate-900 shadow-2xl">
    <header class="border-b border-slate-800 bg-slate-950 px-4 py-3">
      <h1 class="text-base font-semibold text-rose-400">Browser Server</h1>
      <p class="mt-1 text-xs text-slate-400">{{ stats }}</p>
    </header>

    <nav class="grid grid-cols-2 border-b border-slate-800 bg-slate-950/60">
      <button :class="tabHistoryClass" type="button" @click="activePanel = 'history'">History</button>
      <button :class="tabTodosClass" type="button" @click="activePanel = 'todos'">Todos</button>
    </nav>

    <HistoryPanel
      v-show="activePanel === 'history'"
      ref="historyPanel"
      @stats="updateStats"
    />
    <TodosPanel
      v-show="activePanel === 'todos'"
      ref="todosPanel"
      @stats="updateStats"
    />

    <footer class="border-t border-slate-800 bg-slate-950/70 px-4 py-3">
      <div v-show="activePanel === 'history'" class="flex gap-2">
        <button
          type="button"
          class="flex-1 rounded-md border border-slate-700 px-3 py-2 text-sm text-slate-300 hover:border-slate-500 hover:text-white"
          @click="refreshHistory"
        >
          Refresh
        </button>
        <button
          type="button"
          class="flex-1 rounded-md border border-rose-800 px-3 py-2 text-sm text-rose-300 hover:bg-rose-500 hover:text-white"
          @click="clearHistory"
        >
          Clear All
        </button>
      </div>
      <div v-show="activePanel === 'todos'" class="flex gap-2">
        <button
          type="button"
          class="flex-1 rounded-md border border-slate-700 px-3 py-2 text-sm text-slate-300 hover:border-slate-500 hover:text-white"
          @click="refreshTodos"
        >
          Refresh
        </button>
        <button
          type="button"
          class="flex-1 rounded-md border border-rose-800 px-3 py-2 text-sm text-rose-300 hover:bg-rose-500 hover:text-white"
          @click="clearTodos"
        >
          Clear All
        </button>
      </div>
    </footer>
  </main>
</template>
