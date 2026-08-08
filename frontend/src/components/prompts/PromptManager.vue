<template>
  <PromptManagerShell :open="open" @close="requestClose">
    <PromptManagerHeader
      :view="view"
      :title="draft?.title || ''"
      :subtitle="activeTagLabel"
      :count="visiblePrompts.length"
      :is-dirty="isDirty"
      :search="search"
      @update:search="search = $event"
      @create="createPrompt"
      @close="requestClose"
    />

    <!-- Body: sidebar + resizer + main panel -->
    <div ref="containerRef" class="flex min-h-0 flex-1 overflow-hidden">
      <PromptTagSidebar
        class="hidden md:flex"
        :tags="allTags"
        :active-tag="activeTag"
        :total-count="prompts.length"
        :untagged-count="untaggedCount"
        :width="sidebarWidth"
        @select="selectTag"
      />

      <!-- Resizer -->
      <div
        class="hidden w-1 shrink-0 cursor-col-resize bg-slate-200 transition hover:bg-indigo-400 md:block dark:bg-slate-800"
        :class="{ '!bg-indigo-500': isResizing }"
        @pointerdown="startResize"
        @pointermove="onResize"
        @pointerup="stopResize"
        @pointercancel="stopResize"
      ></div>

      <!-- Main panel -->
      <section class="flex min-w-0 flex-1 flex-col overflow-hidden">
        <!-- Grid view -->
        <template v-if="view === 'grid'">
          <PromptToolbar
            :active-tag-label="activeTagLabel"
            :search="search"
            :sort-by="sortBy"
            :layout="layout"
            @update:search="search = $event"
            @update:sort-by="sortBy = $event as typeof sortBy"
            @update:layout="layout = $event"
          />
          <PromptGrid
            :prompts="visiblePrompts"
            :loading="isLoading"
            :layout="layout"
            :search="search"
            :copied-id="copiedId"
            @open="openEditor"
            @use="usePrompt"
            @copy="copyPrompt"
            @delete="confirmDeletePrompt"
            @create="createPrompt"
          />
        </template>

        <!-- Editor view -->
        <PromptEditor
          v-else-if="draft"
          :draft="draft"
          :tags-input="tagsInput"
          :parsed-tags="parsedTags"
          :can-save="canSave"
          :is-saving="isSaving"
          @update:title="draft.title = $event"
          @update:description="draft.description = $event"
          @update:tags-input="tagsInput = $event"
          @update:content="draft.content = $event"
          @save="savePrompt"
          @back="backToGrid"
          @delete="confirmDeletePrompt(draft)"
        />
      </section>
    </div>
  </PromptManagerShell>
</template>

<script setup lang="ts">
import type { PromptResponse } from '../../types'
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { usePrompts } from '../../composables/usePrompts'
import { usePromptManager } from '../../composables/usePromptManager'
import { useResizableSidebar } from '../../composables/useResizableSidebar'
import PromptGrid from './PromptGrid.vue'
import PromptEditor from './PromptEditor.vue'
import PromptToolbar from './PromptToolbar.vue'
import PromptTagSidebar from './PromptTagSidebar.vue'
import PromptManagerShell from './PromptManagerShell.vue'
import PromptManagerHeader from './PromptManagerHeader.vue'

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

const {
  view,
  search,
  sortBy,
  layout,
  copiedId,
  isSaving,
  draft,
  tagsInput,
  parsedTags,
  isDirty,
  canSave,
  activeTagLabel,
  visiblePrompts,
  confirmDiscard,
  resetBrowseState,
  openEditor,
  createPrompt,
  backToGrid,
  savePrompt,
  confirmDeletePrompt,
  copyPrompt,
  selectTag,
} = usePromptManager({ prompts, activeTag, addPrompt, editPrompt, removePrompt, loadPrompts })

const { containerRef, sidebarWidth, isResizing, startResize, onResize, stopResize } =
  useResizableSidebar({
    storageKey: 'pm.sidebarWidth',
  })

/* Always fetch the full list; filtering happens client-side so tag counts stay
   accurate and switching tags is instant. */
watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      resetBrowseState()
      activeTag.value = null
      loadPrompts(null)
    }
  },
  { immediate: true },
)

function usePrompt(prompt: PromptResponse) {
  emit('select', prompt)
  emit('close')
}

function requestClose() {
  if (view.value === 'editor' && !confirmDiscard()) return
  emit('close')
}

/* Cmd/Ctrl+S saves while editing */
function onKeydown(e: KeyboardEvent) {
  if (!props.open) return
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's' && view.value === 'editor') {
    e.preventDefault()
    savePrompt()
  }
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>
