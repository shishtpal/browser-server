<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { faviconUrl, timeAgo } from '../lib/format'
import {
  createApiClient,
  useExtensionSettings,
  useHistoryView,
  useUserId,
} from '../composables/composables'

const emit = defineEmits<{ (event: 'stats', label: string): void }>()

const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const { grouped, stats, errorMessage, load } = useHistoryView(client, userId)

defineExpose({ refresh: load })

onMounted(() => {
  void load()
})

function emitStats() {
  emit('stats', stats.value)
}

function onErrorRetry() {
  void load()
}

function onRefresh() {
  void load()
}
</script>

<template>
  <section class="max-h-[392px] overflow-y-auto" @vue:updated="emitStats">
    <p v-if="errorMessage" class="border-b border-slate-800 px-4 py-3 text-center text-xs text-rose-300">
      {{ errorMessage }}
    </p>
    <div v-else-if="grouped.length === 0" class="px-4 py-10 text-center text-sm text-slate-500">
      No history yet. Browse pages to start tracking.
    </div>
    <ul v-else class="divide-y divide-slate-800">
      <li
        v-for="entry in grouped"
        :key="entry.url"
        class="flex items-center gap-3 px-4 py-3 hover:bg-slate-800/40"
      >
        <img
          :src="faviconUrl(entry.url)"
          alt=""
          class="h-4 w-4 shrink-0 rounded-sm"
          @error="($event.target as HTMLImageElement).style.display = 'none'"
        />
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium text-slate-100" :title="entry.title">{{ entry.title }}</p>
          <p class="truncate text-xs text-slate-500" :title="entry.url">{{ entry.url }}</p>
          <p class="mt-1 text-[11px] text-slate-500">{{ timeAgo(entry.lastVisited) }}</p>
        </div>
        <span class="rounded-full bg-rose-500/10 px-2.5 py-1 text-xs font-semibold text-rose-300">
          {{ entry.count }}
        </span>
      </li>
    </ul>
  </section>
</template>
