<template>
  <Modal
    :open="open"
    title="Attachment Library"
    description="Browse every uploaded image and reuse one in this conversation."
    fullscreen
    @close="$emit('close')"
  >
    <div class="flex h-full flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="mb-4 flex shrink-0 flex-wrap items-center gap-2.5">
        <div class="relative min-w-0 flex-1 sm:min-w-56">
          <svg
            class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-white/35"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M21 21l-4.35-4.35M11 18a7 7 0 100-14 7 7 0 000 14z"
            />
          </svg>
          <input
            v-model="query"
            type="search"
            placeholder="Filter by filename…"
            class="w-full rounded-xl border border-white/10 bg-white/[0.04] py-2 pr-3 pl-9 text-sm text-white transition outline-none placeholder:text-white/35 hover:border-white/20 focus:border-indigo-400/50 focus:bg-white/[0.06] focus:ring-4 focus:ring-indigo-500/10"
          />
        </div>
        <span
          class="shrink-0 rounded-full border border-white/10 bg-white/[0.04] px-2.5 py-1 text-[0.7rem] font-medium text-white/50 tabular-nums"
        >
          {{ filtered.length }}<span v-if="query.trim()"> of {{ attachments.length }}</span>
          {{ attachments.length === 1 ? 'image' : 'images' }}
        </span>
        <button
          type="button"
          class="inline-flex shrink-0 items-center gap-1.5 rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-xs font-semibold text-white/70 transition hover:border-white/20 hover:bg-white/10 hover:text-white active:scale-95 disabled:opacity-40"
          :disabled="loading"
          title="Refresh"
          aria-label="Refresh attachments"
          @click="load"
        >
          <svg
            class="h-3.5 w-3.5"
            :class="{ 'animate-spin': loading }"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.582m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          Refresh
        </button>
      </div>

      <!-- Body -->
      <div class="library-scroll min-h-0 flex-1 overflow-y-auto pr-2">
        <!-- Loading skeleton -->
        <div
          v-if="loading && attachments.length === 0"
          class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6"
        >
          <div
            v-for="i in 12"
            :key="i"
            class="overflow-hidden rounded-xl border border-white/[0.06]"
          >
            <div class="aspect-square animate-pulse bg-white/[0.05]"></div>
            <div class="space-y-2 p-2.5">
              <div class="h-2.5 w-3/4 animate-pulse rounded-full bg-white/[0.08]"></div>
              <div class="h-2 w-1/2 animate-pulse rounded-full bg-white/[0.05]"></div>
            </div>
          </div>
        </div>

        <!-- Error -->
        <div
          v-else-if="error"
          class="flex h-full flex-col items-center justify-center gap-4 py-20 text-center"
        >
          <div
            class="flex h-14 w-14 items-center justify-center rounded-2xl bg-red-500/10 ring-1 ring-red-500/25"
          >
            <svg
              class="h-7 w-7 text-red-400"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M12 9v2m0 4h.01M5.07 19h13.86c1.54 0 2.5-1.67 1.73-3L13.73 4a2 2 0 00-3.46 0L3.34 16c-.77 1.33.19 3 1.73 3z"
              />
            </svg>
          </div>
          <div class="space-y-1">
            <p class="text-sm font-semibold text-white/90">Couldn't load attachments</p>
            <p class="max-w-sm text-xs leading-relaxed text-white/45">{{ error }}</p>
          </div>
          <button
            type="button"
            class="rounded-xl bg-indigo-500 px-4 py-2 text-xs font-semibold text-white shadow-lg shadow-indigo-500/25 transition hover:bg-indigo-400 active:scale-95"
            @click="load"
          >
            Try again
          </button>
        </div>

        <!-- Empty -->
        <div
          v-else-if="attachments.length === 0"
          class="flex h-full flex-col items-center justify-center gap-3 py-20 text-center"
        >
          <div
            class="flex h-16 w-16 items-center justify-center rounded-2xl border border-dashed border-white/15 bg-white/[0.03]"
          >
            <svg
              class="h-8 w-8 text-white/25"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
              />
            </svg>
          </div>
          <p class="text-sm font-semibold text-white/80">No attachments yet</p>
          <p class="max-w-xs text-xs leading-relaxed text-white/40">
            Images you upload to conversations will appear here for reuse.
          </p>
        </div>

        <!-- No matches -->
        <div
          v-else-if="filtered.length === 0"
          class="flex h-full flex-col items-center justify-center gap-3 py-20 text-center"
        >
          <div
            class="flex h-12 w-12 items-center justify-center rounded-2xl bg-white/[0.04] ring-1 ring-white/10"
          >
            <svg
              class="h-6 w-6 text-white/30"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M21 21l-4.35-4.35M11 18a7 7 0 100-14 7 7 0 000 14z"
              />
            </svg>
          </div>
          <p class="text-sm font-medium text-white/70">No matches for “{{ query }}”</p>
          <button
            type="button"
            class="text-xs font-semibold text-indigo-400 transition hover:text-indigo-300"
            @click="query = ''"
          >
            Clear search
          </button>
        </div>

        <!-- Grid -->
        <div
          v-else
          class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6"
        >
          <figure
            v-for="att in filtered"
            :key="att.id"
            class="group relative flex flex-col overflow-hidden rounded-xl border border-white/[0.08] bg-white/[0.03] transition-all duration-200 hover:-translate-y-0.5 hover:border-indigo-400/30 hover:shadow-xl hover:shadow-black/40"
          >
            <button
              type="button"
              class="relative block aspect-square w-full overflow-hidden bg-black/30"
              :title="att.filename"
              @click="openPreview(att)"
            >
              <img
                v-if="!broken.has(att.id)"
                :src="imageUrl(att)"
                :alt="att.filename"
                class="h-full w-full object-cover transition-transform duration-300 ease-out group-hover:scale-105"
                loading="lazy"
                @error="markBroken(att.id)"
              />
              <div
                v-else
                class="flex h-full w-full flex-col items-center justify-center gap-1.5 text-white/25"
              >
                <svg
                  class="h-8 w-8"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                  />
                </svg>
                <span class="text-[0.6rem] font-medium">Preview unavailable</span>
              </div>
            </button>

            <!-- Hover overlay (clicks pass through except on the Reuse button) -->
            <div
              class="pointer-events-none absolute inset-x-0 top-0 flex aspect-square flex-col justify-between bg-gradient-to-t from-black/70 via-black/10 to-transparent opacity-0 transition-opacity duration-200 group-hover:opacity-100"
            >
              <div class="flex justify-end p-2">
                <span
                  class="flex h-7 w-7 items-center justify-center rounded-lg bg-black/50 text-white/80 ring-1 ring-white/15 backdrop-blur-sm"
                >
                  <svg
                    class="h-3.5 w-3.5"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    viewBox="0 0 24 24"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M4 8V6a2 2 0 012-2h2m8 0h2a2 2 0 012 2v2m0 8v2a2 2 0 01-2 2h-2m-8 0H6a2 2 0 01-2-2v-2"
                    />
                  </svg>
                </span>
              </div>
              <div class="flex justify-center p-2.5">
                <button
                  type="button"
                  class="pointer-events-auto inline-flex translate-y-1 items-center gap-1.5 rounded-lg bg-indigo-500 px-3 py-1.5 text-xs font-semibold text-white shadow-lg shadow-black/40 transition-all duration-200 group-hover:translate-y-0 hover:bg-indigo-400 active:scale-95"
                  @click="$emit('reuse', att)"
                >
                  <svg
                    class="h-3.5 w-3.5"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2.5"
                    viewBox="0 0 24 24"
                  >
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                  </svg>
                  Reuse
                </button>
              </div>
            </div>

            <figcaption class="flex min-w-0 flex-col gap-1 border-t border-white/[0.06] p-2.5">
              <div v-if="editingId === att.id" class="flex min-w-0 items-center gap-1.5">
                <input
                  :ref="setRenameInputEl"
                  v-model="editingValue"
                  type="text"
                  maxlength="200"
                  class="min-w-0 flex-1 rounded-lg border border-indigo-400/40 bg-black/50 px-2 py-1 text-xs text-white ring-2 ring-indigo-500/15 outline-none placeholder:text-white/40"
                  placeholder="New filename"
                  @keydown.enter.prevent="saveRename(att)"
                  @keydown.esc="cancelRename"
                />
                <button
                  type="button"
                  class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-emerald-500/15 text-emerald-400 transition hover:bg-emerald-500/30 disabled:opacity-40"
                  title="Save name"
                  :disabled="savingId === att.id"
                  @click="saveRename(att)"
                >
                  <svg
                    class="h-3.5 w-3.5"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    viewBox="0 0 24 24"
                  >
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                </button>
                <button
                  type="button"
                  class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-white/[0.06] text-white/60 transition hover:bg-white/15 hover:text-white disabled:opacity-40"
                  title="Cancel"
                  :disabled="savingId === att.id"
                  @click="cancelRename"
                >
                  <svg
                    class="h-3.5 w-3.5"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    viewBox="0 0 24 24"
                  >
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              <div v-else class="flex min-w-0 items-center gap-1">
                <span class="truncate text-xs font-medium text-white/85" :title="att.filename">{{
                  att.filename
                }}</span>
                <button
                  type="button"
                  class="ml-auto flex h-5 w-5 shrink-0 items-center justify-center rounded-md text-white/35 transition hover:bg-white/10 hover:text-white sm:opacity-0 sm:group-hover:opacity-100"
                  title="Rename"
                  @click="startRename(att)"
                >
                  <svg
                    class="h-3 w-3"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    viewBox="0 0 24 24"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                    />
                  </svg>
                </button>
              </div>
              <span class="text-[0.65rem] font-medium text-white/35 tabular-nums">
                {{ formatBytes(att.size_bytes) }} · {{ formatDate(att.created_at) }}
              </span>
              <span
                v-if="editingId === att.id && renameError"
                class="truncate text-[0.65rem] font-medium text-red-400"
              >
                {{ renameError }}
              </span>
            </figcaption>
          </figure>
        </div>
      </div>
    </div>

    <!-- Lightbox preview -->
    <Teleport to="body">
      <Transition name="lightbox">
        <div
          v-if="preview"
          class="fixed inset-0 z-[60] flex items-center justify-center bg-black/85 p-4 backdrop-blur-sm sm:p-8"
          @click.self="preview = null"
        >
          <figure
            class="relative max-h-full max-w-5xl overflow-hidden rounded-2xl bg-neutral-950 shadow-2xl ring-1 ring-white/10"
          >
            <img
              :src="imageUrl(preview)"
              :alt="preview.filename"
              class="max-h-[80vh] max-w-full object-contain"
            />
            <figcaption
              class="absolute inset-x-0 bottom-0 flex flex-wrap items-end justify-between gap-3 bg-gradient-to-t from-black/90 via-black/50 to-transparent px-5 pt-12 pb-5"
            >
              <div class="min-w-0">
                <p class="truncate text-sm font-semibold text-white">{{ preview.filename }}</p>
                <p class="mt-0.5 text-xs text-white/55 tabular-nums">
                  {{ formatBytes(preview.size_bytes) }} · {{ formatDate(preview.created_at) }}
                </p>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <button
                  type="button"
                  class="rounded-lg border border-white/15 bg-white/10 px-3.5 py-2 text-xs font-semibold text-white/90 backdrop-blur transition hover:bg-white/20 active:scale-95"
                  @click="renameFromPreview"
                >
                  Rename
                </button>
                <button
                  type="button"
                  class="rounded-lg bg-indigo-500 px-3.5 py-2 text-xs font-semibold text-white shadow-lg shadow-indigo-500/25 transition hover:bg-indigo-400 active:scale-95"
                  @click="reuse(preview)"
                >
                  Reuse in this chat
                </button>
              </div>
            </figcaption>
            <button
              type="button"
              class="absolute top-3 right-3 flex h-9 w-9 items-center justify-center rounded-full bg-black/50 text-white/75 ring-1 ring-white/10 backdrop-blur-sm transition hover:bg-black/70 hover:text-white active:scale-95"
              aria-label="Close preview"
              @click="preview = null"
            >
              <svg
                class="h-4.5 h-[1.125rem] w-4.5 w-[1.125rem]"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                viewBox="0 0 24 24"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </figure>
        </div>
      </Transition>
    </Teleport>
  </Modal>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { AIAttachmentSummary } from '@browser-server/shared-types'
import Modal from '../ui/Modal.vue'
import { getAIImageAttachmentUrl, listAIAttachments, renameAIImageAttachment } from '../../lib/api'

const props = defineProps<{ open: boolean }>()

const emit = defineEmits<{
  close: []
  reuse: [attachment: AIAttachmentSummary]
}>()

const attachments = ref<AIAttachmentSummary[]>([])
const loading = ref(false)
const error = ref('')
const query = ref('')
const preview = ref<AIAttachmentSummary | null>(null)
const broken = ref<Set<string>>(new Set())
const editingId = ref<string | null>(null)
const editingValue = ref('')
const savingId = ref<string | null>(null)
const renameError = ref('')
const renameInputEl = ref<HTMLInputElement | null>(null)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return attachments.value
  return attachments.value.filter((a) => a.filename.toLowerCase().includes(q))
})

watch(
  () => props.open,
  (open) => {
    if (open) load()
    else {
      preview.value = null
      cancelRename()
    }
  },
  { immediate: true },
)

// Close the lightbox with Escape
function onPreviewKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') preview.value = null
}
watch(preview, (val) => {
  if (val) window.addEventListener('keydown', onPreviewKeydown)
  else window.removeEventListener('keydown', onPreviewKeydown)
})
onBeforeUnmount(() => window.removeEventListener('keydown', onPreviewKeydown))

async function load() {
  loading.value = true
  error.value = ''
  broken.value = new Set()
  try {
    attachments.value = await listAIAttachments(200)
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load attachments'
  } finally {
    loading.value = false
  }
}

function imageUrl(att: AIAttachmentSummary): string {
  return getAIImageAttachmentUrl(att.conversation_id, att.id)
}

function openPreview(att: AIAttachmentSummary) {
  preview.value = att
}

function reuse(att: AIAttachmentSummary) {
  preview.value = null
  emit('reuse', att)
}

function markBroken(id: string) {
  broken.value = new Set(broken.value).add(id)
}

function setRenameInputEl(el: unknown) {
  renameInputEl.value = el instanceof HTMLInputElement ? el : null
}

async function startRename(att: AIAttachmentSummary) {
  editingId.value = att.id
  editingValue.value = att.filename
  renameError.value = ''
  await nextTick()
  renameInputEl.value?.focus()
  renameInputEl.value?.select()
}

function cancelRename() {
  editingId.value = null
  editingValue.value = ''
  renameError.value = ''
  savingId.value = null
}

async function saveRename(att: AIAttachmentSummary) {
  const name = editingValue.value.trim()
  if (!name) {
    renameError.value = 'Name is required'
    return
  }
  if (name === att.filename) {
    cancelRename()
    return
  }
  savingId.value = att.id
  renameError.value = ''
  try {
    const updated = await renameAIImageAttachment(att.conversation_id, att.id, name)
    att.filename = updated.filename
    if (preview.value && preview.value.id === att.id) preview.value.filename = updated.filename
    cancelRename()
  } catch (err) {
    renameError.value = err instanceof Error ? err.message : 'Failed to rename attachment'
  } finally {
    savingId.value = null
  }
}

function renameFromPreview() {
  const att = preview.value
  if (!att) return
  preview.value = null
  startRename(att)
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function formatDate(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}
</script>

<style scoped>
/* Slim, subtle scrollbar for the grid */
.library-scroll::-webkit-scrollbar {
  width: 8px;
}
.library-scroll::-webkit-scrollbar-track {
  background: transparent;
}
.library-scroll::-webkit-scrollbar-thumb {
  background: rgb(255 255 255 / 0.1);
  border-radius: 999px;
}
.library-scroll::-webkit-scrollbar-thumb:hover {
  background: rgb(255 255 255 / 0.18);
}

/* Lightbox enter/leave animation */
.lightbox-enter-active,
.lightbox-leave-active {
  transition: opacity 0.18s ease;
}
.lightbox-enter-from,
.lightbox-leave-to {
  opacity: 0;
}
.lightbox-enter-active figure {
  transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1);
}
.lightbox-enter-from figure {
  transform: translateY(12px) scale(0.96);
}
</style>
