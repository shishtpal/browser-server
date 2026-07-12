<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { createApiClient, useExtensionSettings, useTodosView, useUserId } from '../composables/composables'
import type { PanelStatus } from './types'

const emit = defineEmits<{ (event: 'status', status: PanelStatus): void }>()

const title = ref('')
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
  init,
  refresh,
  add,
  toggle,
  remove,
  clearAll,
} = useTodosView(client, userId, autoCapture)

defineExpose({
  refresh: () => void refresh(),
})

const isReady = computed(() => Boolean(client.value) && userId.value > 0)
const showSkeleton = computed(() => isLoading.value && items.value.length === 0)

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
  await add(title.value)
  title.value = ''
}
</script>

<template>
  <section class="flex flex-col">
    <!-- Add form -->
    <div class="border-b border-slate-800 px-3 py-3">
      <form class="flex gap-2" @submit.prevent="submit">
        <input
          v-model="title"
          type="text"
          :placeholder="currentDomain ? `Add todo for ${currentDomain}…` : 'Add a todo…'"
          class="flex-1 rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 outline-none transition placeholder:text-slate-500 focus:border-rose-400 focus:ring-2 focus:ring-rose-500/20"
        />
        <button
          type="submit"
          :disabled="!title.trim()"
          class="flex items-center gap-1 rounded-lg bg-rose-500 px-3 py-2 text-sm font-medium text-white transition hover:bg-rose-400 disabled:cursor-not-allowed disabled:opacity-40"
        >
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 5v14M5 12h14" />
          </svg>
        </button>
      </form>

      <div v-if="screenshotPreview" class="mt-2 flex items-center gap-2 rounded-lg bg-slate-800/50 p-2">
        <img :src="screenshotPreview" alt="Screenshot preview" class="h-10 w-16 rounded border border-slate-700 object-cover" />
        <span class="text-[11px] text-slate-400">Screenshot will attach to the next todo</span>
      </div>
    </div>

    <!-- Toolbar -->
    <div v-if="items.length > 0" class="flex items-center justify-between px-3 py-2 text-[11px] text-slate-500">
      <span class="tabular-nums">{{ completed }} of {{ total }} done</span>
      <button
        type="button"
        class="flex items-center gap-1 rounded px-1.5 py-0.5 text-rose-400 transition hover:bg-rose-500/10 hover:text-rose-300"
        @click="clearAll"
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

      <!-- List -->
      <ul v-else class="space-y-0.5">
        <li
          v-for="todo in items"
          :key="todo.id"
          class="group flex items-center gap-3 rounded-lg px-2 py-2.5 transition hover:bg-slate-800/60"
        >
          <input
            type="checkbox"
            class="h-4 w-4 shrink-0 cursor-pointer accent-rose-500"
            :checked="todo.completed"
            @change="toggle(todo.id, ($event.target as HTMLInputElement).checked)"
          />
          <img
            v-if="todo.screenshotUrl"
            :src="todo.screenshotUrl"
            alt="Screenshot"
            class="h-8 w-12 shrink-0 rounded border border-slate-700 object-cover"
          />
          <div class="min-w-0 flex-1">
            <p
              class="truncate text-sm font-medium"
              :class="todo.completed ? 'text-slate-500 line-through' : 'text-slate-100'"
              :title="todo.title"
            >
              {{ todo.title }}
            </p>
            <p class="text-[10px] text-slate-500">{{ todo.updatedAtLabel }}</p>
          </div>
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
  </section>
</template>
