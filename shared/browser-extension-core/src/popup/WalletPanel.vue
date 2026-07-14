<script setup lang="ts">
import { computed, watch } from 'vue'
import { createApiClient, useExtensionSettings, useUserId, useWalletView } from '../composables/composables'
import type { PanelStatus } from './types'
import WalletAddForm from './wallet/WalletAddForm.vue'
import WalletItemCard from './wallet/WalletItemCard.vue'

const emit = defineEmits<{ (event: 'status', status: PanelStatus): void }>()

const { settings } = useExtensionSettings()
const userId = useUserId(computed(() => settings.value))
const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

const { currentDomain, items, total, isLoading, errorMessage, init, refresh, create, reveal, update } = useWalletView(client, userId)

async function handleAdd(payload: { login_provider: string; username: string; password: string; description?: string }) {
  if (!currentDomain.value) return
  await create({ website: currentDomain.value, ...payload })
}

defineExpose({
  refresh: () => void refresh(),
})

const isReady = computed(() => Boolean(client.value) && userId.value > 0)
const showSkeleton = computed(() => isLoading.value && items.value.length === 0)

watch(
  [isReady, isLoading, errorMessage, total],
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
</script>

<template>
  <section class="px-2 py-2">
    <!-- Skeleton -->
    <div v-if="showSkeleton" class="space-y-2 px-1">
      <div v-for="n in 3" :key="n" class="space-y-2 rounded-lg border border-slate-800 p-3">
        <div class="h-3 w-2/3 animate-pulse rounded bg-slate-800" />
        <div class="h-2.5 w-1/2 animate-pulse rounded bg-slate-800/70" />
        <div class="h-2.5 w-1/3 animate-pulse rounded bg-slate-800/70" />
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="errorMessage" class="flex flex-col items-center gap-3 px-4 py-12 text-center">
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
    <div v-else-if="!currentDomain" class="flex flex-col items-center gap-2 px-4 py-12 text-center">
      <p class="text-sm font-medium text-slate-300">No active site</p>
      <p class="max-w-[260px] text-xs text-slate-500">Open a regular web page to see saved credentials for it.</p>
    </div>

    <template v-else>
      <WalletAddForm
        :domain="currentDomain"
        :has-items="items.length > 0"
        @add="handleAdd"
      />

      <p v-if="items.length === 0" class="px-4 py-5 text-center text-xs text-slate-500">No saved accounts yet. Add the first one above.</p>

      <!-- List -->
      <ul v-else class="space-y-1.5">
        <WalletItemCard
          v-for="item in items"
          :key="item.id"
          :item="item"
          :reveal-fn="reveal"
          :update-fn="update"
        />
      </ul>
    </template>
  </section>
</template>
