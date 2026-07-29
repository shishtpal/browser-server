<template>
  <Modal
    :open="open"
    title="Prompt Manager"
    description="Organise prompts with tags on the left, browse on the right"
    fullscreen
    @close="requestClose"
  >
    <div class="flex h-full min-h-0 flex-col overflow-hidden rounded-2xl border border-white/10 bg-slate-950/70 text-slate-100 shadow-2xl shadow-black/30">
      <!-- ───────────────────────── Header ───────────────────────── -->
      <header class="flex flex-wrap items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
        <div class="min-w-0">
          <div class="flex items-center gap-2 text-sm font-semibold">
            <span class="truncate">{{ view === 'editor' ? (draft?.title || 'Untitled prompt') : 'Prompt library' }}</span>
            <span v-if="view === 'editor' && isDirty" class="rounded-full bg-amber-400/15 px-2 py-0.5 text-[0.65rem] font-semibold uppercase tracking-wide text-amber-300">
              Unsaved
            </span>
          </div>
          <div class="truncate text-xs text-slate-400">
            {{ activeTagLabel }}
            <template v-if="view === 'grid'"> · {{ visiblePrompts.length }} prompt{{ visiblePrompts.length === 1 ? '' : 's' }}</template>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <div v-if="view === 'grid'" class="relative">
            <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-xs text-slate-500">🔍</span>
            <input
              v-model="search"
              type="search"
              placeholder="Search prompts…"
              class="w-56 rounded-lg border border-white/10 bg-slate-900/80 py-1.5 pl-8 pr-3 text-sm text-slate-100 outline-none placeholder:text-slate-500 focus:border-cyan-400"
            />
          </div>

          <button
            class="rounded-lg border border-cyan-400/30 bg-cyan-500/15 px-3 py-1.5 text-sm font-medium text-cyan-200 hover:bg-cyan-500/25"
            type="button"
            @click="createPrompt"
          >
            + New prompt
          </button>
          <button
            class="rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-sm font-medium text-slate-200 hover:bg-white/10"
            type="button"
            @click="requestClose"
          >
            Close
          </button>
        </div>
      </header>

      <!-- ───────────────────────── Body ───────────────────────── -->
      <div ref="containerRef" class="flex min-h-0 flex-1 overflow-hidden">
        <!-- Sidebar: tags -->
        <aside
          class="flex min-w-[200px] shrink-0 flex-col overflow-hidden border-r border-white/10 bg-slate-900/80"
          :style="{ width: `${sidebarWidth}px` }"
        >
          <div class="flex items-center justify-between px-4 py-3">
            <h3 class="text-sm font-semibold text-slate-100">Tags</h3>
          </div>

          <nav class="flex-1 space-y-1 overflow-auto px-2 pb-4">
            <!-- All prompts -->
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm transition"
              :class="activeTag === null ? 'bg-cyan-500/15 text-cyan-200' : 'text-slate-300 hover:bg-white/5'"
              @click="selectTag(null)"
            >
              <span>🏷️</span>
              <span class="flex-1 truncate">All prompts</span>
              <span class="text-xs text-slate-500">{{ prompts.length }}</span>
            </button>

            <!-- Untagged -->
            <button
              v-if="untaggedCount > 0"
              type="button"
              class="flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm transition"
              :class="activeTag === UNTAGGED_FILTER ? 'bg-cyan-500/15 text-cyan-200' : 'text-slate-300 hover:bg-white/5'"
              @click="selectTag(UNTAGGED_FILTER)"
            >
              <span>📄</span>
              <span class="flex-1 truncate">Untagged</span>
              <span class="text-xs text-slate-500">{{ untaggedCount }}</span>
            </button>

            <div class="my-2 border-t border-white/5"></div>

            <p v-if="allTags.length === 0" class="rounded-lg border border-dashed border-white/10 px-3 py-4 text-center text-xs text-slate-500">
              No tags yet
            </p>

            <button
              v-for="tag in allTags"
              :key="tag.name"
              type="button"
              class="flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm transition"
              :class="activeTag === tag.name ? 'bg-cyan-500/15 text-cyan-200' : 'text-slate-300 hover:bg-white/5'"
              @click="selectTag(tag.name)"
            >
              <span>#</span>
              <span class="flex-1 truncate">{{ tag.name }}</span>
              <span class="text-xs text-slate-500">{{ tag.count }}</span>
            </button>
          </nav>
        </aside>

        <!-- Resizer -->
        <div
          class="w-1.5 shrink-0 cursor-col-resize bg-slate-800/70 transition hover:bg-cyan-500/60"
          :class="{ 'bg-cyan-500/60': isResizing }"
          @pointerdown="startResize"
          @pointermove="onResize"
          @pointerup="stopResize"
          @pointercancel="stopResize"
        ></div>

        <!-- Main panel -->
        <section class="flex min-w-0 flex-1 flex-col overflow-hidden bg-slate-950/90">
          <!-- ══════════ GRID VIEW (default) ══════════ -->
          <template v-if="view === 'grid'">
            <div class="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-2.5">
              <div class="flex items-center gap-2 text-xs text-slate-400">
                <span class="font-semibold text-slate-200">{{ activeTagLabel }}</span>
                <span v-if="search">· results for “{{ search }}”</span>
              </div>
              <div class="flex items-center gap-2">
                <select
                  v-model="sortBy"
                  class="rounded-lg border border-white/10 bg-slate-900/80 px-2 py-1.5 text-xs text-slate-200 outline-none focus:border-cyan-400"
                >
                  <option value="updated">Recently updated</option>
                  <option value="created">Recently created</option>
                  <option value="title">Title A→Z</option>
                </select>
                <div class="flex overflow-hidden rounded-lg border border-white/10">
                  <button
                    type="button"
                    class="px-2 py-1.5 text-xs"
                    :class="layout === 'grid' ? 'bg-white/10 text-slate-100' : 'text-slate-400 hover:bg-white/5'"
                    title="Grid view"
                    @click="layout = 'grid'"
                  >▦</button>
                  <button
                    type="button"
                    class="px-2 py-1.5 text-xs"
                    :class="layout === 'list' ? 'bg-white/10 text-slate-100' : 'text-slate-400 hover:bg-white/5'"
                    title="List view"
                    @click="layout = 'list'"
                  >☰</button>
                </div>
              </div>
            </div>

            <div class="flex-1 overflow-auto px-4 py-4">
              <div v-if="isLoading" class="grid gap-4" :class="gridClass">
                <div v-for="n in 6" :key="n" class="h-40 animate-pulse rounded-2xl border border-white/5 bg-white/5"></div>
              </div>

              <div
                v-else-if="visiblePrompts.length === 0"
                class="flex h-full flex-col items-center justify-center gap-3 rounded-2xl border border-dashed border-white/10 bg-white/5 p-10 text-center"
              >
                <div class="text-3xl">🪄</div>
                <p class="text-sm text-slate-300">
                  {{ search ? 'No prompts match your search.' : 'No prompts match this tag yet.' }}
                </p>
                <button
                  class="rounded-lg bg-cyan-500 px-3 py-2 text-sm font-semibold text-slate-950 hover:bg-cyan-400"
                  type="button"
                  @click="createPrompt"
                >
                  Create your first prompt
                </button>
              </div>

              <div v-else class="grid gap-4" :class="gridClass">
                <article
                  v-for="prompt in visiblePrompts"
                  :key="prompt.id"
                  class="group relative flex cursor-pointer flex-col rounded-2xl border border-white/10 bg-slate-900/60 p-4 text-left transition hover:-translate-y-0.5 hover:border-cyan-400/40 hover:bg-slate-900 hover:shadow-lg hover:shadow-cyan-500/5"
                  role="button"
                  tabindex="0"
                  @click="openEditor(prompt)"
                  @keydown.enter.prevent="openEditor(prompt)"
                  @keydown.space.prevent="openEditor(prompt)"
                >
                  <!-- hover actions -->
                  <div class="absolute right-3 top-3 flex gap-1 opacity-0 transition group-hover:opacity-100">
                    <button
                      class="rounded-md border border-white/10 bg-slate-950/80 px-1.5 py-1 text-xs text-slate-300 hover:text-cyan-300"
                      type="button"
                      title="Use this prompt"
                      @click.stop="usePrompt(prompt)"
                    >➤</button>
                    <button
                      class="rounded-md border border-white/10 bg-slate-950/80 px-1.5 py-1 text-xs text-slate-300 hover:text-cyan-300"
                      type="button"
                      title="Copy content"
                      @click.stop="copyPrompt(prompt)"
                    >{{ copiedId === prompt.id ? '✓' : '⧉' }}</button>
                    <button
                      class="rounded-md border border-white/10 bg-slate-950/80 px-1.5 py-1 text-xs text-red-400 hover:text-red-300"
                      type="button"
                      title="Delete prompt"
                      @click.stop="confirmDeletePrompt(prompt)"
                    >🗑️</button>
                  </div>

                  <h4 class="pr-20 text-sm font-semibold text-slate-100">
                    {{ prompt.title || 'Untitled prompt' }}
                  </h4>
                  <p v-if="prompt.description" class="mt-1 line-clamp-2 text-xs text-slate-400">
                    {{ prompt.description }}
                  </p>
                  <p class="mt-3 line-clamp-4 whitespace-pre-wrap rounded-lg bg-slate-950/60 p-2.5 text-[0.72rem] leading-5 text-slate-400">
                    {{ prompt.content }}
                  </p>

                  <div class="mt-auto flex flex-wrap items-center gap-1 pt-3">
                    <span
                      v-for="tag in (prompt.tags || []).slice(0, 4)"
                      :key="tag"
                      class="rounded-full bg-white/10 px-2 py-0.5 text-[0.65rem] text-slate-300"
                    >{{ tag }}</span>
                    <span v-if="(prompt.tags?.length || 0) > 4" class="text-[0.65rem] text-slate-500">
                      +{{ (prompt.tags?.length || 0) - 4 }}
                    </span>
                  </div>

                  <div class="mt-2 flex items-center justify-end border-t border-white/5 pt-2 text-[0.65rem] text-slate-500">
                    <span>{{ formatDate(prompt.updated_at || prompt.created_at) }}</span>
                  </div>
                </article>
              </div>
            </div>
          </template>

          <!-- ══════════ EDITOR VIEW ══════════ -->
          <template v-else>
            <div class="flex items-center justify-between border-b border-white/10 px-4 py-2.5">
              <button
                class="flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-sm font-medium text-slate-300 hover:bg-white/10"
                type="button"
                @click="backToGrid"
              >
                ← Back to prompts
              </button>
              <span class="text-xs text-slate-500">
                {{ draft?.id ? `Editing · last saved ${formatDate(draft?.updated_at)}` : 'New prompt' }}
              </span>
            </div>

            <form v-if="draft" class="flex min-h-0 flex-1 flex-col overflow-auto px-4 py-4" @submit.prevent="savePrompt">
              <div class="mx-auto flex w-full max-w-4xl flex-1 flex-col gap-4">
                <div>
                  <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-400">Title</label>
                  <input
                    ref="titleInputRef"
                    v-model="draft.title"
                    class="w-full rounded-xl border border-white/10 bg-slate-900/80 px-3 py-2 text-sm text-slate-100 outline-none focus:border-cyan-400"
                    placeholder="Prompt title"
                    required
                  />
                </div>

                <div>
                  <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-400">Description</label>
                  <input
                    v-model="draft.description"
                    class="w-full rounded-xl border border-white/10 bg-slate-900/80 px-3 py-2 text-sm text-slate-100 outline-none focus:border-cyan-400"
                    placeholder="Short description"
                  />
                </div>

                <div>
                  <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-400">Tags</label>
                  <input
                    v-model="tagsInput"
                    class="w-full rounded-xl border border-white/10 bg-slate-900/80 px-3 py-2 text-sm text-slate-100 outline-none focus:border-cyan-400"
                    placeholder="system, assistant, coding"
                  />
                  <div v-if="parsedTags.length" class="mt-2 flex flex-wrap gap-1">
                    <span v-for="tag in parsedTags" :key="tag" class="rounded-full bg-white/10 px-2 py-0.5 text-[0.7rem] text-slate-300">
                      {{ tag }}
                    </span>
                  </div>
                </div>

                <div class="flex min-h-[320px] flex-1 flex-col">
                  <div class="mb-1 flex items-center justify-between">
                    <label class="block text-xs font-semibold uppercase tracking-wide text-slate-400">Prompt content</label>
                    <span class="text-[0.7rem] text-slate-500">{{ draft.content.length }} chars</span>
                  </div>
                  <textarea
                    v-model="draft.content"
                    class="h-full min-h-[320px] flex-1 w-full resize-none rounded-2xl border border-white/10 bg-slate-900/80 px-3 py-3 font-mono text-sm leading-6 text-slate-100 outline-none focus:border-cyan-400"
                    placeholder="Write or edit your prompt here…"
                    required
                  />
                </div>
              </div>
            </form>

            <div class="flex items-center justify-between border-t border-white/10 bg-slate-900/70 px-4 py-3">
              <button
                v-if="draft?.id"
                class="rounded-lg px-3 py-2 text-sm font-semibold text-red-400 hover:bg-red-500/10"
                type="button"
                @click="confirmDeletePrompt(draft)"
              >
                Delete prompt
              </button>
              <span v-else></span>

              <div class="flex items-center gap-2">
                <button
                  class="rounded-lg border border-white/10 px-3 py-2 text-sm font-semibold text-slate-300 hover:bg-white/10"
                  type="button"
                  @click="backToGrid"
                >
                  Cancel
                </button>
                <button
                  class="rounded-lg bg-cyan-500 px-4 py-2 text-sm font-semibold text-slate-950 transition hover:bg-cyan-400 disabled:cursor-not-allowed disabled:opacity-50"
                  type="button"
                  :disabled="!canSave || isSaving"
                  @click="savePrompt"
                >
                  {{ isSaving ? 'Saving…' : 'Save prompt' }}
                </button>
              </div>
            </div>
          </template>
        </section>
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Modal from '../ui/Modal.vue'
import type {
  PromptResponse,
  CreatePromptInput,
  UpdatePromptInput,
} from '../../types'
import { UNTAGGED_FILTER, usePrompts, type TagFilter } from '../../composables/usePrompts'

const props = defineProps<{
  open: boolean
  userId: number | null
}>()

const emit = defineEmits<{
  close: []
  select: [prompt: PromptResponse]
}>()

const userIdRef = computed(() => props.userId)

const {
  prompts,
  isLoading,
  allTags,
  untaggedCount,
  activeTag,
  loadPrompts,
  addPrompt,
  editPrompt,
  removePrompt,
} = usePrompts(userIdRef)

/* ─────────────── view state ─────────────── */
type ViewMode = 'grid' | 'editor'

const view = ref<ViewMode>('grid')
const search = ref('')
const sortBy = ref<'updated' | 'created' | 'title'>('updated')
const layout = ref<'grid' | 'list'>('grid')
const copiedId = ref<number | null>(null)
const isSaving = ref(false)

/* Always fetch the full list; filtering happens client-side so tag
   counts stay accurate and switching tags is instant. */
async function refresh() {
  activeTag.value = null
  await loadPrompts(null)
}

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      view.value = 'grid'
      activeTag.value = null
      search.value = ''
      draft.value = null
      refresh()
    }
  },
  { immediate: true },
)

/* ─────────────── tags ─────────────── */
const activeTagLabel = computed(() => {
  if (activeTag.value === null) return 'All prompts'
  if (activeTag.value === UNTAGGED_FILTER) return 'Untagged'
  return activeTag.value
})

function selectTag(tag: TagFilter) {
  activeTag.value = tag
  if (view.value === 'editor' && !confirmDiscard()) return
  view.value = 'grid'
  draft.value = null
}

/* ─────────────── grid data ─────────────── */
const visiblePrompts = computed(() => {
  let list = prompts.value.slice()
  const selectedTag = activeTag.value

  if (selectedTag === UNTAGGED_FILTER) {
    list = list.filter(p => !(p.tags?.length))
  } else if (typeof selectedTag === 'string') {
    list = list.filter(p => (p.tags || []).includes(selectedTag))
  }

  const q = search.value.trim().toLowerCase()
  if (q) {
    list = list.filter(p =>
      (p.title || '').toLowerCase().includes(q) ||
      (p.description || '').toLowerCase().includes(q) ||
      (p.content || '').toLowerCase().includes(q) ||
      (p.tags || []).some(t => t.toLowerCase().includes(q)),
    )
  }

  return list.sort((a, b) => {
    if (sortBy.value === 'title') return (a.title || '').localeCompare(b.title || '')
    const key = sortBy.value === 'created' ? 'created_at' : 'updated_at'
    return new Date(b[key] || b.created_at || 0).getTime() - new Date(a[key] || a.created_at || 0).getTime()
  })
})

const gridClass = computed(() =>
  layout.value === 'list'
    ? 'grid-cols-1'
    : 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4',
)

function formatDate(value?: string | null) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}

async function copyPrompt(prompt: PromptResponse) {
  try {
    await navigator.clipboard.writeText(prompt.content || '')
    copiedId.value = prompt.id
    setTimeout(() => { if (copiedId.value === prompt.id) copiedId.value = null }, 1500)
  } catch { /* clipboard unavailable */ }
}

function usePrompt(prompt: PromptResponse) {
  emit('select', prompt)
  emit('close')
}

/* ─────────────── editor ─────────────── */
interface PromptDraft {
  id: number
  title: string
  content: string
  description: string
  updated_at?: string | null
  tags?: string[]
}

const draft = ref<PromptDraft | null>(null)
const tagsInput = ref('')
const snapshot = ref('')
const titleInputRef = ref<HTMLInputElement | null>(null)

const parsedTags = computed(() =>
  tagsInput.value.split(',').map(t => t.trim()).filter(Boolean),
)

const isDirty = computed(() => {
  if (!draft.value) return false
  return JSON.stringify({ ...draft.value, tags: parsedTags.value }) !== snapshot.value
})

const canSave = computed(() =>
  !!draft.value && draft.value.title.trim().length > 0 && draft.value.content.trim().length > 0,
)

function takeSnapshot() {
  snapshot.value = JSON.stringify({ ...draft.value, tags: parsedTags.value })
}

function openEditor(prompt: PromptResponse) {
  draft.value = {
    id: prompt.id,
    title: prompt.title || '',
    content: prompt.content || '',
    description: prompt.description || '',
    updated_at: prompt.updated_at,
  }
  tagsInput.value = (prompt.tags || []).join(', ')
  takeSnapshot()
  view.value = 'editor'
  nextTick(() => titleInputRef.value?.focus())
}

function createPrompt() {
  draft.value = {
    id: 0,
    title: '',
    content: '',
    description: '',
  }
  tagsInput.value = activeTag.value && activeTag.value !== UNTAGGED_FILTER ? activeTag.value : ''
  takeSnapshot()
  view.value = 'editor'
  nextTick(() => titleInputRef.value?.focus())
}

function confirmDiscard() {
  if (!isDirty.value) return true
  return window.confirm('You have unsaved changes. Discard them?')
}

function backToGrid() {
  if (!confirmDiscard()) return
  draft.value = null
  tagsInput.value = ''
  view.value = 'grid'
}

async function savePrompt() {
  if (!draft.value || !canSave.value || isSaving.value) return
  isSaving.value = true
  try {
    const payload: CreatePromptInput | UpdatePromptInput = {
      title: draft.value.title.trim(),
      content: draft.value.content,
      description: draft.value.description || undefined,
      tags: parsedTags.value,
    }

    if (draft.value.id) {
      await editPrompt(draft.value.id, payload as UpdatePromptInput)
    } else {
      await addPrompt(payload as CreatePromptInput)
    }

    await loadPrompts(null)
    draft.value = null
    tagsInput.value = ''
    view.value = 'grid'
  } finally {
    isSaving.value = false
  }
}

async function confirmDeletePrompt(prompt: PromptResponse | PromptDraft) {
  if (!prompt.id) return
  if (!window.confirm(`Delete prompt "${prompt.title || 'Untitled'}"?`)) return
  await removePrompt(prompt.id)
  await loadPrompts(null)
  if (draft.value?.id === prompt.id) {
    draft.value = null
    tagsInput.value = ''
    view.value = 'grid'
  }
}

/* ─────────────── close guard ─────────────── */
function requestClose() {
  if (view.value === 'editor' && !confirmDiscard()) return
  emit('close')
}

/* ─────────────── resizable sidebar ─────────────── */
const containerRef = ref<HTMLElement | null>(null)
const sidebarWidth = ref(Number(localStorage.getItem('pm.sidebarWidth')) || 260)
const isResizing = ref(false)

function startResize(event: PointerEvent) {
  isResizing.value = true
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
  event.preventDefault()
}

function onResize(event: PointerEvent) {
  if (!isResizing.value || !containerRef.value) return
  const rect = containerRef.value.getBoundingClientRect()
  const next = event.clientX - rect.left
  sidebarWidth.value = Math.min(Math.max(next, 200), Math.max(240, rect.width - 420))
}

function stopResize(event: PointerEvent) {
  if (!isResizing.value) return
  isResizing.value = false
  try { (event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId) } catch { /* noop */ }
  localStorage.setItem('pm.sidebarWidth', String(Math.round(sidebarWidth.value)))
}

/* ─────────────── shortcuts ─────────────── */
function onKeydown(e: KeyboardEvent) {
  if (!props.open) return
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') {
    if (view.value === 'editor') {
      e.preventDefault()
      savePrompt()
    }
  }
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.line-clamp-2,
.line-clamp-4 {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.line-clamp-2 { 
  line-clamp: 2;
  -webkit-line-clamp: 2; 
}
.line-clamp-4 {
  line-clamp: 4; 
  -webkit-line-clamp: 4; 
}
</style>
