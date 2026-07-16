<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { createApiClient, useExtensionSettings, useTodosView, useUserId } from '../composables/composables'
import type { PanelStatus } from './types'

const emit = defineEmits<{ (event: 'status', status: PanelStatus): void }>()

const title = ref('')
const description = ref('')
const filter = ref<'all' | 'active' | 'completed'>('all')
const editingId = ref<number | null>(null)
const editTitle = ref('')
const editDescription = ref('')
const screenshot = ref<{ url: string; title: string } | null>(null)
const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))
const autoCapture = computed(() => Boolean(settings.value?.autoCapture))

const {
  currentDomain,
  screenshotPreview,
  items,
  total,
  completed,
  isLoading,
  errorMessage,
  actionError,
  init,
  refresh,
  add,
  update,
  toggle,
  remove,
  clearAll,
} = useTodosView(client, userId, autoCapture)

defineExpose({
  refresh: () => void refresh(),
})

const isReady = computed(() => Boolean(client.value) && userId.value > 0)
const showSkeleton = computed(() => isLoading.value && items.value.length === 0)
const visibleItems = computed(() => items.value.filter((todo) => {
  if (filter.value === 'active') return !todo.completed
  if (filter.value === 'completed') return todo.completed
  return true
}))

watch(
  [isReady, isLoading, errorMessage, total, completed],
  () => {
    emit('status', {
      count: total.value,
      state: errorMessage.value ? 'error' : isLoading.value ? 'loading' : 'ready',
    })
  },
  { immediate: true },
)

// Auto-load once the API client is ready (settings load asynchronously).
watch(
  [client, userId],
  () => {
    if (isReady.value) void init()
  },
  { immediate: true },
)

async function submit() {
  if (!title.value.trim()) return
  if (await add(title.value, description.value)) {
    title.value = ''
    description.value = ''
  }
}

function startEdit(todo: (typeof items.value)[number]) {
  editingId.value = todo.id
  editTitle.value = todo.title
  editDescription.value = todo.description
}

async function saveEdit(id: number) {
  if (!editTitle.value.trim()) return
  if (await update(id, { title: editTitle.value, description: editDescription.value })) {
    editingId.value = null
  }
}

function confirmClearAll() {
  if (window.confirm(`Delete all ${total.value} todos for ${currentDomain.value}?`)) void clearAll()
}
</script>

<template>
  <section class="flex flex-col">
    <!-- Add form -->
    <div class="border-b border-slate-800 px-3 py-3">
      <form class="flex flex-col gap-2" @submit.prevent="submit">
        <div class="flex gap-2">
          <input
            v-model="title"
            type="text"
            :placeholder="currentDomain ? `Add todo for ${currentDomain}…` : 'Add a todo…'"
            class="flex-1 rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 outline-none transition placeholder:text-slate-500 focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
          />
          <button
            type="submit"
            :disabled="!title.trim() || !currentDomain"
            class="flex items-center gap-1 rounded-lg bg-rose-500 px-3 py-2 text-sm font-medium text-white transition hover:bg-rose-400 disabled:cursor-not-allowed disabled:opacity-40"
            title="Add todo"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M12 5v14M5 12h14" />
            </svg>
          </button>
        </div>
        <textarea
          v-model="description"
          rows="2"
          placeholder="Details (optional)"
          class="w-full resize-none rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-xs text-slate-100 outline-none transition placeholder:text-slate-500 focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
        />
      </form>

      <div v-if="screenshotPreview" class="mt-2 flex items-center gap-2 rounded-lg bg-slate-800/50 p-2">
        <img :src="screenshotPreview" alt="Screenshot preview" class="h-10 w-16 rounded border border-slate-700 object-cover" />
        <span class="text-[11px] text-slate-400">Screenshot will attach to the next todo</span>
      </div>
    </div>

    <p v-if="actionError" class="border-b border-rose-500/20 bg-rose-500/10 px-3 py-2 text-xs text-rose-300">{{ actionError }}</p>

    <!-- Toolbar -->
    <div v-if="items.length > 0" class="flex items-center justify-between gap-2 px-3 py-2 text-[11px] text-slate-500">
      <div class="flex items-center gap-1">
        <button
          v-for="option in (['all', 'active', 'completed'] as const)"
          :key="option"
          type="button"
          class="rounded px-1.5 py-0.5 capitalize transition"
          :class="filter === option ? 'bg-rose-500/15 text-rose-300' : 'hover:bg-slate-800 hover:text-slate-300'"
          @click="filter = option"
        >{{ option }}</button>
      </div>
      <span class="ml-auto tabular-nums">{{ completed }}/{{ total }} done</span>
      <button
        type="button"
        class="flex items-center gap-1 rounded px-1.5 py-0.5 text-rose-400 transition hover:bg-rose-500/10 hover:text-rose-300"
        @click="confirmClearAll"
      >
        <svg class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
        </svg>
        Clear all
      </button>
    </div>

    <div class="px-2 pb-2">
      <!-- Skeleton -->
      <div v-if="showSkeleton" class="space-y-1 pt-1">
        <div v-for="n in 4" :key="n" class="flex items-center gap-3 rounded-lg px-2 py-2.5">
          <div class="h-4 w-4 shrink-0 animate-pulse rounded bg-slate-800" />
          <div class="h-3 flex-1 animate-pulse rounded bg-slate-800" />
        </div>
      </div>

      <!-- Error -->
      <div v-else-if="errorMessage" class="flex flex-col items-center gap-3 px-4 py-10 text-center">
        <div class="flex h-12 w-12 items-center justify-center rounded-full bg-rose-500/10">
          <svg class="h-6 w-6 text-rose-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 9v4M12 17h.01" />
            <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          </svg>
        </div>
        <p class="text-sm font-medium text-slate-300">Can't reach the server</p>
        <p class="max-w-[280px] text-xs text-slate-500">{{ errorMessage }}</p>
        <button type="button" class="rounded-lg bg-rose-500 px-4 py-1.5 text-xs font-medium text-white transition hover:bg-rose-400" @click="refresh">
          Try again
        </button>
      </div>

      <!-- No domain -->
      <div v-else-if="!currentDomain" class="flex flex-col items-center gap-2 px-4 py-10 text-center">
        <p class="text-sm font-medium text-slate-300">No active site</p>
        <p class="max-w-[260px] text-xs text-slate-500">Open a regular web page to add page-specific todos.</p>
      </div>

      <!-- Empty -->
      <div v-else-if="items.length === 0" class="flex flex-col items-center gap-3 px-4 py-10 text-center">
        <div class="flex h-12 w-12 items-center justify-center rounded-full bg-slate-800">
          <svg class="h-6 w-6 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 11l3 3L22 4" />
            <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
          </svg>
        </div>
        <p class="text-sm font-medium text-slate-300">No todos for this site</p>
        <p class="max-w-[260px] text-xs text-slate-500">Add one above to keep track of tasks for {{ currentDomain }}.</p>
      </div>

      <div v-else-if="visibleItems.length === 0" class="px-4 py-8 text-center text-xs text-slate-500">
        No {{ filter }} todos for this site.
      </div>

      <!-- List -->
      <ul v-else class="space-y-0.5">
        <li
          v-for="todo in visibleItems"
          :key="todo.id"
          class="group flex items-start gap-3 rounded-lg px-2 py-2.5 transition hover:bg-slate-800/60"
        >
          <input
            type="checkbox"
            class="mt-1 h-4 w-4 shrink-0 cursor-pointer accent-rose-500"
            :checked="todo.completed"
            @change="toggle(todo.id, ($event.target as HTMLInputElement).checked)"
          />
          <button
            v-if="todo.screenshotUrl"
            type="button"
            class="mt-0.5 shrink-0 cursor-zoom-in transition hover:opacity-80"
            title="View screenshot"
            @click="screenshot = { url: todo.screenshotUrl!, title: todo.title }"
          >
            <img :src="todo.screenshotUrl" alt="Screenshot" class="h-8 w-12 rounded border border-slate-700 object-cover" />
          </button>
          <div class="min-w-0 flex-1">
            <form v-if="editingId === todo.id" class="space-y-1.5" @submit.prevent="saveEdit(todo.id)">
              <input v-model="editTitle" class="w-full rounded border border-slate-600 bg-slate-950 px-2 py-1 text-xs text-slate-100 outline-none focus:border-rose-400" />
              <textarea v-model="editDescription" rows="2" placeholder="Details (optional)" class="w-full resize-none rounded border border-slate-600 bg-slate-950 px-2 py-1 text-xs text-slate-100 outline-none focus:border-rose-400" />
              <div class="flex gap-1">
                <button type="submit" :disabled="!editTitle.trim()" class="rounded bg-rose-500 px-2 py-1 text-[10px] font-medium text-white disabled:opacity-40">Save</button>
                <button type="button" class="rounded px-2 py-1 text-[10px] text-slate-400 hover:bg-slate-700" @click="editingId = null">Cancel</button>
              </div>
            </form>
            <template v-else>
              <p class="break-words text-sm font-medium" :class="todo.completed ? 'text-slate-500 line-through' : 'text-slate-100'">{{ todo.title }}</p>
              <p v-if="todo.description" class="mt-0.5 whitespace-pre-wrap break-words text-xs leading-4 text-slate-400">{{ todo.description }}</p>
              <p class="mt-1 text-[10px] text-slate-500" :title="`Created ${todo.createdAtLabel}`">Updated {{ todo.updatedAtLabel }} · {{ todo.domain }}</p>
            </template>
          </div>
          <button
            v-if="editingId !== todo.id"
            type="button"
            title="Edit"
            class="flex h-7 w-7 shrink-0 items-center justify-center rounded text-slate-500 opacity-0 transition hover:bg-slate-700 hover:text-white group-hover:opacity-100"
            @click="startEdit(todo)"
          >
            <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>
          </button>
          <button
            type="button"
            title="Delete"
            class="flex h-7 w-7 shrink-0 items-center justify-center rounded text-slate-500 opacity-0 transition hover:bg-rose-500 hover:text-white group-hover:opacity-100"
            @click="remove(todo.id)"
          >
            <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
            </svg>
          </button>
        </li>
      </ul>
    </div>

    <div v-if="screenshot" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/90 p-4" @click.self="screenshot = null">
      <div class="flex max-h-full w-full flex-col gap-2">
        <div class="flex items-center justify-between gap-3">
          <p class="truncate text-sm font-medium text-slate-100">{{ screenshot.title }}</p>
          <button type="button" class="rounded px-2 py-1 text-xs text-slate-300 hover:bg-slate-800" @click="screenshot = null">Close</button>
        </div>
        <img :src="screenshot.url" :alt="screenshot.title" class="max-h-[420px] w-full rounded-lg border border-slate-700 object-contain" />
      </div>
    </div>
  </section>
</template>
