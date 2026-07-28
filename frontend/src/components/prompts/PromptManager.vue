<template>
  <Modal :open="open" title="Prompt Manager" description="Browse folders and edit prompts in a full-screen workspace" fullscreen @close="$emit('close')">
    <div class="flex h-full min-h-0 flex-col overflow-hidden rounded-2xl border border-white/10 bg-slate-950/70 text-slate-100 shadow-2xl shadow-black/30">
      <div class="flex items-center justify-between border-b border-white/10 px-4 py-3">
        <div>
          <div class="text-sm font-semibold">{{ editingPrompt?.title || 'Prompt workspace' }}</div>
          <div class="text-xs text-slate-400">{{ selectedFolderName || 'All prompts' }}</div>
        </div>
        <div class="flex items-center gap-2">
          <button class="rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-sm font-medium text-slate-200 hover:bg-white/10" type="button" @click="createPrompt">
            + New prompt
          </button>
          <button class="rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-sm font-medium text-slate-200 hover:bg-white/10" type="button" @click="$emit('close')">
            Close
          </button>
        </div>
      </div>

      <div ref="containerRef" class="flex min-h-0 flex-1 overflow-hidden">
        <aside class="flex min-w-[220px] flex-col overflow-hidden bg-slate-900/80" :style="{ width: `${sidebarWidth}px` }">
          <div class="border-b border-white/10 px-4 py-3">
            <div class="mb-2 flex items-center justify-between">
              <h3 class="text-sm font-semibold text-slate-100">Folders</h3>
              <button class="rounded-lg px-2 py-1 text-xs font-semibold text-slate-300 hover:bg-white/10" type="button" @click="createFolder">+ New</button>
            </div>
            <div v-if="folders.length === 0" class="rounded-lg border border-dashed border-white/10 px-3 py-4 text-center text-xs text-slate-400">
              No folders yet
            </div>
            <ul v-else class="space-y-1">
              <li v-for="folder in folders" :key="folder.id" class="flex items-center rounded-lg px-2 py-1.5 hover:bg-white/5">
                <button class="flex-1 truncate text-left text-sm text-slate-300" type="button" @click="selectFolder(folder.id)">
                  📁 {{ folder.name }}
                </button>
                <button class="rounded px-1.5 py-1 text-xs text-slate-400 hover:text-slate-200" type="button" @click="startRenameFolder(folder)">✏️</button>
                <button class="rounded px-1.5 py-1 text-xs text-red-400 hover:text-red-300" type="button" @click="confirmDeleteFolder(folder)">🗑️</button>
              </li>
            </ul>
          </div>

          <div class="flex-1 overflow-auto px-4 py-3">
            <div class="mb-2 flex items-center justify-between">
              <h3 class="text-sm font-semibold text-slate-100">{{ selectedFolderName || 'All prompts' }}</h3>
              <span class="text-xs text-slate-500">{{ filteredPrompts.length }}</span>
            </div>
            <div v-if="isLoading" class="py-8 text-center text-sm text-slate-500">Loading…</div>
            <div v-else-if="filteredPrompts.length === 0" class="rounded-lg border border-dashed border-white/10 px-3 py-6 text-center text-xs text-slate-500">
              No prompts in this view
            </div>
            <ul v-else class="space-y-2">
              <li v-for="prompt in filteredPrompts" :key="prompt.id">
                <button class="w-full rounded-xl border border-white/10 bg-slate-950/60 p-3 text-left transition hover:border-cyan-400/40 hover:bg-slate-800/80" type="button" @click="startEditPrompt(prompt)">
                  <div class="text-sm font-semibold text-slate-100">{{ prompt.title || 'Untitled prompt' }}</div>
                  <div v-if="prompt.description" class="mt-1 truncate text-xs text-slate-400">{{ prompt.description }}</div>
                  <div v-if="prompt.tags?.length" class="mt-2 flex flex-wrap gap-1">
                    <span v-for="tag in prompt.tags" :key="tag" class="rounded-full bg-white/10 px-2 py-0.5 text-[0.7em] text-slate-300">
                      {{ tag }}
                    </span>
                  </div>
                </button>
              </li>
            </ul>
          </div>
        </aside>

        <div class="w-2 cursor-col-resize bg-slate-800/70 hover:bg-cyan-500/60" @pointerdown="startResize" @pointerup="stopResize"></div>

        <section class="flex flex-1 flex-col overflow-hidden bg-slate-950/90">
          <div class="flex-1 overflow-auto px-4 py-4">
            <div v-if="!editingPrompt" class="flex h-full items-center justify-center rounded-2xl border border-dashed border-white/10 bg-white/5 p-8 text-center text-sm text-slate-400">
              Select a prompt from the left pane or create a new one to start editing it here.
            </div>
            <form v-else class="mx-auto flex h-full max-w-4xl flex-col gap-4" @submit.prevent="savePrompt">
              <div class="grid gap-4 md:grid-cols-[1.2fr_0.8fr]">
                <div>
                  <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-400">Title</label>
                  <input v-model="editingPrompt.title" class="w-full rounded-xl border border-white/10 bg-slate-900/80 px-3 py-2 text-sm text-slate-100 outline-none ring-0 focus:border-cyan-400" placeholder="Prompt title" required />
                </div>
                <div>
                  <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-400">Folder</label>
                  <select v-model="editingPrompt.folder_id" class="w-full rounded-xl border border-white/10 bg-slate-900/80 px-3 py-2 text-sm text-slate-100 outline-none focus:border-cyan-400">
                    <option :value="null">None</option>
                    <option v-for="folder in folders" :key="folder.id" :value="folder.id">{{ folder.name }}</option>
                  </select>
                </div>
              </div>

              <div>
                <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-400">Description</label>
                <input v-model="editingPrompt.description" class="w-full rounded-xl border border-white/10 bg-slate-900/80 px-3 py-2 text-sm text-slate-100 outline-none focus:border-cyan-400" placeholder="Short description" />
              </div>

              <div>
                <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-400">Tags</label>
                <input v-model="editingPromptTags" class="w-full rounded-xl border border-white/10 bg-slate-900/80 px-3 py-2 text-sm text-slate-100 outline-none focus:border-cyan-400" placeholder="system, assistant, coding" />
              </div>

              <div class="flex-1">
                <label class="mb-1 block text-xs font-semibold uppercase tracking-wide text-slate-400">Prompt content</label>
                <textarea v-model="editingPrompt.content" class="h-full min-h-[320px] w-full rounded-2xl border border-white/10 bg-slate-900/80 px-3 py-3 text-sm leading-6 text-slate-100 outline-none focus:border-cyan-400" placeholder="Write or edit your prompt here..." required />
              </div>
            </form>
          </div>

          <div v-if="editingPrompt" class="flex items-center justify-between border-t border-white/10 bg-slate-900/70 px-4 py-3">
            <button class="rounded-lg px-3 py-2 text-sm font-semibold text-red-400 hover:bg-red-500/10" type="button" @click="confirmDeletePrompt(editingPrompt)">
              Delete prompt
            </button>
            <div class="flex gap-2">
              <button class="rounded-lg border border-white/10 px-3 py-2 text-sm font-semibold text-slate-300 hover:bg-white/10" type="button" @click="editingPrompt = null">
                Clear
              </button>
              <button class="rounded-lg bg-cyan-500 px-3 py-2 text-sm font-semibold text-slate-950 hover:bg-cyan-400" type="submit" @click="savePrompt">
                Save prompt
              </button>
            </div>
          </div>
        </section>
      </div>
    </div>

    <Modal v-if="editingFolder" :open="true" title="Rename folder" @close="editingFolder = null">
      <form @submit.prevent="saveRenameFolder">
        <input v-model="editingFolderName" class="w-full rounded-lg border border-white/10 bg-slate-900/80 px-3 py-2 text-sm text-slate-100 outline-none focus:border-cyan-400" placeholder="Folder name" autofocus />
        <div class="mt-4 flex justify-end gap-2">
          <button class="rounded-lg px-4 py-2 text-sm font-semibold text-slate-300 hover:bg-white/10" type="button" @click="editingFolder = null">Cancel</button>
          <button class="rounded-lg bg-cyan-500 px-4 py-2 text-sm font-semibold text-slate-950" type="submit">Save</button>
        </div>
      </form>
    </Modal>
  </Modal>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Modal from '../ui/Modal.vue'
import type { PromptFolder, PromptResponse, CreatePromptFolderInput, UpdatePromptFolderInput, CreatePromptInput, UpdatePromptInput } from '../../types'
import { usePrompts } from '../../composables/usePrompts'

const props = defineProps<{
  open: boolean
  userId: number | null
}>()

const emit = defineEmits<{
  close: []
  select: [prompt: PromptResponse]
}>()

const {
  folders,
  prompts,
  isLoading,
  selectedFolderId,
  loadFolders,
  loadPrompts,
  addFolder,
  renameFolder,
  removeFolder,
  addPrompt,
  editPrompt,
  removePrompt,
} = usePrompts(ref(props.userId))

watch(() => props.open, (val) => {
  if (val) {
    loadFolders()
    loadPrompts(null)
  }
})

const selectedFolderName = computed(() => {
  if (!selectedFolderId.value) return 'All prompts'
  const f = folders.value.find(x => x.id === selectedFolderId.value)
  return f?.name ?? 'All prompts'
})

const filteredPrompts = computed(() => {
  if (!selectedFolderId.value) return prompts.value
  return prompts.value.filter(p => p.folder_id === selectedFolderId.value)
})

function selectFolder(id: number | null) {
  selectedFolderId.value = id
  loadPrompts(id)
}

function createFolder() {
  const name = window.prompt('Folder name')
  if (!name?.trim()) return
  addFolder({ user_id: props.userId ?? 0, name: name.trim() })
}

const editingFolder = ref<PromptFolder | null>(null)
const editingFolderName = ref('')

function startRenameFolder(folder: PromptFolder) {
  editingFolder.value = folder
  editingFolderName.value = folder.name
}

async function saveRenameFolder() {
  if (!editingFolder.value || !editingFolderName.value.trim()) return
  await renameFolder(editingFolder.value.id, { name: editingFolderName.value.trim() })
  editingFolder.value = null
  editingFolderName.value = ''
}

async function confirmDeleteFolder(folder: PromptFolder) {
  if (!window.confirm(`Delete folder "${folder.name}"? Prompts will be moved to unfiled.`)) return
  await removeFolder(folder.id)
  if (selectedFolderId.value === folder.id) selectedFolderId.value = null
}

const editingPrompt = ref<(PromptResponse & { folder_id: number | null | undefined }) | null>(null)
const editingPromptTags = ref('')
const containerRef = ref<HTMLElement | null>(null)
const sidebarWidth = ref(320)
const isResizing = ref(false)

function createPrompt() {
  editingPrompt.value = {
    id: 0,
    user_id: props.userId ?? 0,
    folder_id: selectedFolderId.value,
    title: '',
    content: '',
    description: '',
    tags: [],
    created_at: '',
    updated_at: '',
    folder_name: null,
  }
  editingPromptTags.value = ''
}

function startEditPrompt(prompt: PromptResponse) {
  editingPrompt.value = { ...prompt, folder_id: prompt.folder_id ?? null }
  editingPromptTags.value = (prompt.tags || []).join(', ')
}

async function savePrompt() {
  if (!editingPrompt.value) return

  const data: CreatePromptInput | UpdatePromptInput = {
    title: editingPrompt.value.title,
    content: editingPrompt.value.content,
    description: editingPrompt.value.description || undefined,
    folder_id: editingPrompt.value.folder_id ?? null,
    tags: editingPromptTags.value.split(',').map(t => t.trim()).filter(Boolean),
  }

  if (editingPrompt.value.id) {
    await editPrompt(editingPrompt.value.id, data as UpdatePromptInput)
  } else {
    await addPrompt(data as CreatePromptInput)
  }

  editingPrompt.value = null
  editingPromptTags.value = ''
}

async function confirmDeletePrompt(prompt: PromptResponse) {
  if (!window.confirm(`Delete prompt "${prompt.title}"?`)) return
  await removePrompt(prompt.id)
  if (editingPrompt.value?.id === prompt.id) {
    editingPrompt.value = null
    editingPromptTags.value = ''
  }
}

function startResize(event: MouseEvent) {
  isResizing.value = true
  event.preventDefault()
}

function stopResize() {
  isResizing.value = false
}

function onResize(event: MouseEvent) {
  if (!isResizing.value || !containerRef.value) return
  const rect = containerRef.value.getBoundingClientRect()
  const nextWidth = Math.min(Math.max(event.clientX - rect.left, 220), rect.width - 320)
  sidebarWidth.value = Math.max(220, Math.min(nextWidth, rect.width - 320))
}

onMounted(() => {
  window.addEventListener('mousemove', onResize)
  window.addEventListener('mouseup', stopResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', onResize)
  window.removeEventListener('mouseup', stopResize)
})
</script>
