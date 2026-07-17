<script setup lang="ts">
import type { GroupedHistoryEntry } from '@browser-server/shared-client'
import { computed, onBeforeUnmount, ref } from 'vue'
import { faviconUrl } from '@browser-server/shared-utils'
import { getBrowserApi } from '../browserApi'
import { toEmbedUrl } from '../lib/embedUrl'
import { hasDnrSupport, installUnsafePreviewRule, removeUnsafePreviewRule } from '../lib/unsafePreview'

const props = defineProps<{
  entry: GroupedHistoryEntry | null
  unsafePreviewEnabled: boolean
  resizing: boolean
}>()

const unsafePreviewActive = ref(false)
const dnrAvailable = computed(() => hasDnrSupport())

const previewUrl = computed(() => {
  if (!props.entry) return ''
  return toEmbedUrl(props.entry.url)
})

async function toggleUnsafePreview() {
  unsafePreviewActive.value = !unsafePreviewActive.value

  if (!unsafePreviewActive.value) {
    await removeUnsafePreviewRule()
    return
  }

  if (props.entry) {
    try {
      const hostname = new URL(props.entry.url).hostname
      await installUnsafePreviewRule(hostname)
    } catch {
      // Invalid URL
    }
  }
}

function openInTab(url: string) {
  void getBrowserApi().tabs.create({ url, active: true })
}

/**
 * Called by the parent when the selected entry changes so we can update
 * the DNR rule to match the new hostname.
 */
defineExpose({
  onEntryChanged(entry: GroupedHistoryEntry) {
    if (unsafePreviewActive.value) {
      try {
        const hostname = new URL(entry.url).hostname
        void installUnsafePreviewRule(hostname)
      } catch {
        // Invalid URL
      }
    }
  },
})

onBeforeUnmount(() => {
  void removeUnsafePreviewRule()
})
</script>

<template>
  <section class="flex min-w-0 flex-col overflow-hidden">
    <template v-if="entry">
      <div class="flex h-14 shrink-0 items-center gap-3 border-b border-slate-800 px-4">
        <img :src="faviconUrl(entry.url)" alt="" class="h-4 w-4 rounded-sm" />
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium">{{ entry.title || entry.url }}</p>
          <p class="truncate text-[11px] text-slate-500">{{ entry.url }}</p>
        </div>
        <button
          v-if="unsafePreviewEnabled && dnrAvailable"
          type="button"
          class="shrink-0 rounded-lg px-3 py-1.5 text-xs font-medium transition"
          :class="unsafePreviewActive ? 'bg-amber-600 text-white hover:bg-amber-500' : 'border border-slate-700 text-slate-400 hover:border-amber-500 hover:text-amber-300'"
          :title="unsafePreviewActive ? 'Unsafe preview active — click to disable' : 'Strip anti-framing headers for this preview'"
          @click="toggleUnsafePreview"
        >
          {{ unsafePreviewActive ? '⚠ Unsafe On' : '🔓 Unsafe Preview' }}
        </button>
        <button
          type="button"
          class="shrink-0 rounded-lg bg-rose-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-rose-400"
          @click="openInTab(entry.url)"
        >
          Open in tab
        </button>
      </div>
      <iframe
        :key="previewUrl + (unsafePreviewActive ? '?unsafe' : '')"
        :src="previewUrl"
        :title="`Preview of ${entry.title || entry.url}`"
        sandbox="allow-forms allow-modals allow-popups allow-same-origin allow-scripts"
        class="min-h-0 flex-1 bg-white"
        :class="{ 'pointer-events-none': resizing }"
      />
    </template>
    <div v-else class="flex flex-1 flex-col items-center justify-center gap-3 px-8 text-center">
      <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-900 text-slate-600">
        <svg class="h-7 w-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 3h7v7M10 14 21 3M21 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5" /></svg>
      </div>
      <p class="text-sm font-medium text-slate-300">Select a link for quick view</p>
      <p class="max-w-sm text-xs leading-5 text-slate-600">Some websites prevent embedded previews. Use “Open in tab” when a page does not load here.</p>
      <p v-if="unsafePreviewEnabled && dnrAvailable" class="max-w-sm text-xs leading-5 text-amber-500/80">Unsafe preview mode is enabled — anti-framing headers will be stripped when toggled on.</p>
    </div>
  </section>
</template>
