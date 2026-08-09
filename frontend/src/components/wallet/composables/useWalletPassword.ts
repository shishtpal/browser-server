import { ref, type Ref } from 'vue';

/**
 * Per-entry password reveal/copy state, shared by the table row and the card.
 *
 * Lazily fetches via the provided `reveal` (auth-scoped endpoint), caches for
 * the session of that component, and flashes "copied" feedback on copy.
 */
export function useWalletPassword(reveal: () => Promise<string>) {
  const revealed = ref(false);
  const password: Ref<string | null> = ref(null);
  const revealedPassword = ref('');
  const loading = ref(false);
  const copied = ref(false);
  let copiedTimer: ReturnType<typeof setTimeout> | null = null;

  /** Fetch once and cache; reuses the cached value afterwards. */
  const fetchPassword = async (): Promise<string> => {
    if (password.value !== null) return password.value;
    loading.value = true;
    try {
      password.value = await reveal();
      return password.value;
    } finally {
      loading.value = false;
    }
  };

  const toggleReveal = async () => {
    if (revealed.value) {
      revealed.value = false;
      revealedPassword.value = '';
      return;
    }
    const pw = await fetchPassword();
    revealedPassword.value = pw;
    revealed.value = true;
  };

  const copyPassword = async () => {
    const pw = await fetchPassword();
    try {
      await navigator.clipboard.writeText(pw);
      copied.value = true;
      if (copiedTimer) clearTimeout(copiedTimer);
      copiedTimer = setTimeout(() => (copied.value = false), 1500);
    } catch {
      // Clipboard API unavailable (non-secure context) — silently ignore.
    }
  };

  return { revealed, revealedPassword, loading, copied, toggleReveal, copyPassword };
}
