import { onUnmounted, ref } from 'vue';

/**
 * Persisted, drag-to-resize for the right-hand tools panel (handle on the
 * panel's left edge; width measured from the right). Not the same geometry as
 * the left-edge `useResizableSidebar` in src/composables.
 */
export function usePanelResize(options: {
  storageKey: string;
  min: number;
  max: number;
  initial: number;
}) {
  const panelWidth = ref(loadWidth());
  let startX = 0;
  let startWidth = 0;

  function loadWidth(): number {
    const stored = localStorage.getItem(options.storageKey);
    if (stored) {
      const parsed = Number(stored);
      if (parsed >= options.min && parsed <= options.max) return parsed;
    }
    return options.initial;
  }

  function startResize(e: MouseEvent) {
    e.preventDefault();
    startX = e.clientX;
    startWidth = panelWidth.value;
    document.addEventListener('mousemove', onResize);
    document.addEventListener('mouseup', stopResize);
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  }

  function onResize(e: MouseEvent) {
    const delta = startX - e.clientX;
    panelWidth.value = Math.min(options.max, Math.max(options.min, startWidth + delta));
  }

  function stopResize() {
    document.removeEventListener('mousemove', onResize);
    document.removeEventListener('mouseup', stopResize);
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    localStorage.setItem(options.storageKey, String(panelWidth.value));
  }

  onUnmounted(() => {
    document.removeEventListener('mousemove', onResize);
    document.removeEventListener('mouseup', stopResize);
  });

  return { panelWidth, startResize };
}
