<script setup lang="ts">
import { ref, useTemplateRef, watch } from 'vue'
import HistoryPanel from './HistoryPanel.vue'
import TodosPanel from './TodosPanel.vue'
import WalletPanel from './WalletPanel.vue'

type Panel = 'history' | 'todos' | 'wallet'

const props = defineProps<{ initialPanel: Panel }>()

const activePanel = ref<Panel>(props.initialPanel)
const stats = ref('Loading…')

const historyPanel = useTemplateRef<InstanceType<typeof HistoryPanel>>('historyPanel')
const todosPanel = useTemplateRef<InstanceType<typeof TodosPanel>>('todosPanel')
const walletPanel = useTemplateRef<InstanceType<typeof WalletPanel>>('walletPanel')

function updateStats(label: string) {
  stats.value = label
}

function tabClass(panel: Panel) {
  return activePanel.value === panel
    ? 'border-b-2 border-rose-400 px-3 py-2 text-sm font-medium text-rose-300'
    : 'border-b-2 border-transparent px-3 py-2 text-sm font-medium text-slate-400'
}

function refreshHistory() {
  historyPanel.value?.refresh()
}

function refreshTodos() {
  todosPanel.value?.refresh()
}

function refreshWallet() {
  walletPanel.value?.refresh()
}

function clearTodos() {
  todosPanel.value?.clearAll()
}

watch(activePanel, () => {
  if (activePanel.value === 'history') {
    updateStats('Loading…')
  } else if (activePanel.value === 'todos') {
    updateStats('0 todos · 0 done')
  } else {
    updateStats('0 passwords')
  }
})
</script>

<template>
  <main class="w-[480px] max-h-[540px] overflow-hidden rounded-xl border border-slate-800 bg-slate-900 shadow-2xl">
    <header class="border-b border-slate-800 bg-slate-950 px-4 py-3">
      <h1 class="text-base font-semibold text-rose-400">Browser Server</h1>
      <p class="mt-1 text-xs text-slate-400">{{ stats }}</p>
    </header>

    <nav class="grid grid-cols-3 border-b border-slate-800 bg-slate-950/60">
      <button :class="tabClass('history')" type="button" @click="activePanel = 'history'">History</button>
      <button :class="tabClass('todos')" type="button" @click="activePanel = 'todos'">Todos</button>
      <button :class="tabClass('wallet')" type="button" @click="activePanel = 'wallet'">Wallet</button>
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
    <WalletPanel
      v-show="activePanel === 'wallet'"
      ref="walletPanel"
      @stats="updateStats"
    />

    <footer class="border-t border-slate-800 bg-slate-950/70 px-4 py-3">
      <div v-show="activePanel === 'history'" class="flex gap-2">
        <button
          type="button"
          class="w-full rounded-md border border-slate-700 px-3 py-2 text-sm text-slate-300 hover:border-slate-500 hover:text-white"
          @click="refreshHistory"
        >
          Refresh
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
      <div v-show="activePanel === 'wallet'" class="flex gap-2">
        <button
          type="button"
          class="w-full rounded-md border border-slate-700 px-3 py-2 text-sm text-slate-300 hover:border-slate-500 hover:text-white"
          @click="refreshWallet"
        >
          Refresh
        </button>
      </div>
    </footer>
  </main>
</template>
