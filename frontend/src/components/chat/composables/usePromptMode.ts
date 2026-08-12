import { ref, type Ref } from 'vue';
import { searchPrompts } from '../../../lib/api';
import type { PromptResponse } from '../../../types';

export interface PromptDropdownApi {
  move: (delta: number) => void;
  activate: () => void;
}

/**
 * The "/" prompt-search mode of the chat input: mode state, debounced search,
 * dropdown keyboard delegation, and selection. The textarea value itself stays
 * with the caller (v-model) — this composable only nudges it for prefix toggles.
 */
export function usePromptMode(
  userId: Ref<number | null | undefined>,
  setValue: (v: string) => void,
  getValue: () => string,
) {
  const promptMode = ref(false);
  const promptQuery = ref('');
  const promptResults = ref<PromptResponse[]>([]);
  const promptLoading = ref(false);
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  function clearDebounce() {
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }
  }

  function enterPromptMode(query = '') {
    promptMode.value = true;
    promptQuery.value = query;
    setValue('/' + query);
    if (query.trim()) runSearch(query);
    else {
      promptResults.value = [];
      promptLoading.value = false;
    }
  }

  /** Exit prompt mode but keep the current input text. */
  function exitPromptMode() {
    promptMode.value = false;
    promptQuery.value = '';
    promptResults.value = [];
    promptLoading.value = false;
    clearDebounce();
  }

  /** Exit prompt mode and clear the input (Escape). */
  function clearPromptMode() {
    exitPromptMode();
    setValue('');
  }

  function runSearch(query: string) {
    const uid = userId.value;
    if (!uid || uid <= 0) return;
    const q = query.trim();
    if (!q) {
      promptResults.value = [];
      promptLoading.value = false;
      return;
    }
    promptLoading.value = true;
    searchPrompts(uid, q, 20)
      .then((r) => {
        promptResults.value = r;
      })
      .catch(() => {
        promptResults.value = [];
      })
      .finally(() => {
        promptLoading.value = false;
      });
  }

  function debouncedSearch(query: string) {
    clearDebounce();
    debounceTimer = setTimeout(() => runSearch(query), 180);
  }

  /** Called on every input; drives entering/exiting prompt mode from the text. */
  function onTextChanged() {
    const value = getValue();
    if (promptMode.value) {
      if (!value.startsWith('/')) {
        // Backspaced over the "/" — exit, keep remaining text.
        exitPromptMode();
        return;
      }
      const query = value.slice(1);
      promptQuery.value = query;
      if (!query.trim()) {
        promptResults.value = [];
        promptLoading.value = false;
        return;
      }
      debouncedSearch(query);
      return;
    }
    if (value.startsWith('/')) {
      enterPromptMode(value.slice(1));
    }
  }

  /** Keyboard shortcut handler for prompt mode. Returns true if consumed. */
  function onKeydown(event: KeyboardEvent, dropdown: PromptDropdownApi | null): boolean {
    if (!promptMode.value) return false;
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      dropdown?.move(1);
      return true;
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault();
      dropdown?.move(-1);
      return true;
    }
    if (event.key === 'Enter') {
      event.preventDefault();
      dropdown?.activate();
      return true;
    }
    if (event.key === 'Escape') {
      event.preventDefault();
      clearPromptMode();
      return true;
    }
    return false; // let printable keys reach the textarea so the /query can be typed
  }

  return {
    promptMode,
    promptQuery,
    promptResults,
    promptLoading,
    enterPromptMode,
    exitPromptMode,
    clearPromptMode,
    onTextChanged,
    onKeydown,
    clearDebounce,
  };
}
