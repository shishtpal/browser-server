import type { PromptResponse, CreatePromptInput, UpdatePromptInput } from '../types';
import { computed, ref, type Ref } from 'vue';
import { UNTAGGED_FILTER, type TagFilter } from './usePrompts';

export type PromptView = 'grid' | 'editor';
export type PromptSort = 'updated' | 'created' | 'title';
export type PromptLayout = 'grid' | 'list';

export interface PromptDraft {
  id: number;
  title: string;
  content: string;
  description: string;
  updated_at?: string | null;
}

interface PromptManagerDeps {
  prompts: Ref<PromptResponse[]>;
  activeTag: Ref<TagFilter>;
  addPrompt: (data: CreatePromptInput) => Promise<PromptResponse | undefined>;
  editPrompt: (id: number, data: UpdatePromptInput) => Promise<PromptResponse | undefined>;
  removePrompt: (id: number) => Promise<void>;
  loadPrompts: (query?: string | null) => Promise<void>;
}

/**
 * Orchestrates the Prompt Manager: grid/editor view switching, search + sort +
 * layout state, draft editing with dirty tracking, and persistence actions.
 * All data access is injected so the composable stays free of API concerns.
 */
export function usePromptManager(deps: PromptManagerDeps) {
  const { prompts, activeTag, addPrompt, editPrompt, removePrompt, loadPrompts } = deps;

  /* ── view / browse state ── */
  const view = ref<PromptView>('grid');
  const search = ref('');
  const sortBy = ref<PromptSort>('updated');
  const layout = ref<PromptLayout>('grid');
  const copiedId = ref<number | null>(null);
  const isSaving = ref(false);

  /* ── editor state ── */
  const draft = ref<PromptDraft | null>(null);
  const tagsInput = ref('');
  const snapshot = ref('');

  const parsedTags = computed(() =>
    tagsInput.value
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean),
  );

  const isDirty = computed(() => {
    if (!draft.value) return false;
    return JSON.stringify({ ...draft.value, tags: parsedTags.value }) !== snapshot.value;
  });

  const canSave = computed(
    () =>
      !!draft.value && draft.value.title.trim().length > 0 && draft.value.content.trim().length > 0,
  );

  const activeTagLabel = computed(() => {
    if (activeTag.value === null) return 'All prompts';
    if (activeTag.value === UNTAGGED_FILTER) return 'Untagged';
    return activeTag.value;
  });

  const visiblePrompts = computed(() => {
    let list = prompts.value.slice();
    const selectedTag = activeTag.value;

    if (selectedTag === UNTAGGED_FILTER) {
      list = list.filter((p) => !p.tags?.length);
    } else if (typeof selectedTag === 'string') {
      list = list.filter((p) => (p.tags || []).includes(selectedTag));
    }

    const q = search.value.trim().toLowerCase();
    if (q) {
      list = list.filter(
        (p) =>
          (p.title || '').toLowerCase().includes(q) ||
          (p.description || '').toLowerCase().includes(q) ||
          (p.content || '').toLowerCase().includes(q) ||
          (p.tags || []).some((t) => t.toLowerCase().includes(q)),
      );
    }

    return list.sort((a, b) => {
      if (sortBy.value === 'title') return (a.title || '').localeCompare(b.title || '');
      const key = sortBy.value === 'created' ? 'created_at' : 'updated_at';
      return (
        new Date(b[key] || b.created_at || 0).getTime() -
        new Date(a[key] || a.created_at || 0).getTime()
      );
    });
  });

  /* ── helpers ── */
  function takeSnapshot() {
    snapshot.value = JSON.stringify({ ...draft.value, tags: parsedTags.value });
  }

  function confirmDiscard() {
    if (!isDirty.value) return true;
    return window.confirm('You have unsaved changes. Discard them?');
  }

  function resetBrowseState() {
    view.value = 'grid';
    search.value = '';
    draft.value = null;
    tagsInput.value = '';
  }

  /* ── editor actions ── */
  function openEditor(prompt: PromptResponse) {
    draft.value = {
      id: prompt.id,
      title: prompt.title || '',
      content: prompt.content || '',
      description: prompt.description || '',
      updated_at: prompt.updated_at,
    };
    tagsInput.value = (prompt.tags || []).join(', ');
    takeSnapshot();
    view.value = 'editor';
  }

  function createPrompt() {
    draft.value = { id: 0, title: '', content: '', description: '' };
    tagsInput.value = activeTag.value && activeTag.value !== UNTAGGED_FILTER ? activeTag.value : '';
    takeSnapshot();
    view.value = 'editor';
  }

  function backToGrid() {
    if (!confirmDiscard()) return;
    draft.value = null;
    tagsInput.value = '';
    view.value = 'grid';
  }

  async function savePrompt() {
    if (!draft.value || !canSave.value || isSaving.value) return;
    isSaving.value = true;
    try {
      const payload: CreatePromptInput | UpdatePromptInput = {
        title: draft.value.title.trim(),
        content: draft.value.content,
        description: draft.value.description || undefined,
        tags: parsedTags.value,
      };
      if (draft.value.id) {
        await editPrompt(draft.value.id, payload as UpdatePromptInput);
      } else {
        await addPrompt(payload as CreatePromptInput);
      }
      await loadPrompts(null);
      draft.value = null;
      tagsInput.value = '';
      view.value = 'grid';
    } finally {
      isSaving.value = false;
    }
  }

  async function confirmDeletePrompt(prompt: PromptResponse | PromptDraft) {
    if (!prompt.id) return;
    if (!window.confirm(`Delete prompt "${prompt.title || 'Untitled'}"?`)) return;
    await removePrompt(prompt.id);
    await loadPrompts(null);
    if (draft.value?.id === prompt.id) {
      draft.value = null;
      tagsInput.value = '';
      view.value = 'grid';
    }
  }

  /* ── grid actions ── */
  async function copyPrompt(prompt: PromptResponse) {
    try {
      await navigator.clipboard.writeText(prompt.content || '');
      copiedId.value = prompt.id;
      setTimeout(() => {
        if (copiedId.value === prompt.id) copiedId.value = null;
      }, 1500);
    } catch {
      /* clipboard unavailable */
    }
  }

  function selectTag(tag: TagFilter) {
    if (view.value === 'editor' && !confirmDiscard()) return;
    activeTag.value = tag;
    view.value = 'grid';
    draft.value = null;
    tagsInput.value = '';
  }

  return {
    // state
    view,
    search,
    sortBy,
    layout,
    copiedId,
    isSaving,
    draft,
    tagsInput,
    // derived
    parsedTags,
    isDirty,
    canSave,
    activeTagLabel,
    visiblePrompts,
    // actions
    confirmDiscard,
    resetBrowseState,
    openEditor,
    createPrompt,
    backToGrid,
    savePrompt,
    confirmDeletePrompt,
    copyPrompt,
    selectTag,
  };
}
