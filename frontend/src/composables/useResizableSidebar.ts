import { ref } from 'vue';

export interface ResizableSidebarOptions {
  /** localStorage key used to persist the width across sessions. */
  storageKey?: string;
  /** Minimum sidebar width in pixels. */
  min?: number;
  /** Space (px) reserved for the main panel when computing the max width. */
  reserve?: number;
  /** Fallback width when nothing is persisted. */
  initial?: number;
}

/**
 * Pointer-driven, persisted, horizontally resizable sidebar.
 *
 * Bind `containerRef` to the flex row that holds the sidebar + main panel,
 * apply `sidebarWidth` to the sidebar's width, and wire the pointer handlers
 * to the drag handle element.
 */
export function useResizableSidebar(options: ResizableSidebarOptions = {}) {
  const storageKey = options.storageKey ?? 'pm.sidebarWidth';
  const min = options.min ?? 200;
  const reserve = options.reserve ?? 420;
  const initial = options.initial ?? 260;

  const containerRef = ref<HTMLElement | null>(null);
  const isResizing = ref(false);

  const stored = typeof localStorage !== 'undefined' ? Number(localStorage.getItem(storageKey)) : 0;
  const sidebarWidth = ref(stored > 0 ? stored : initial);

  function startResize(event: PointerEvent) {
    isResizing.value = true;
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    event.preventDefault();
  }

  function onResize(event: PointerEvent) {
    if (!isResizing.value || !containerRef.value) return;
    const rect = containerRef.value.getBoundingClientRect();
    const next = event.clientX - rect.left;
    sidebarWidth.value = Math.min(Math.max(next, min), Math.max(min + 40, rect.width - reserve));
  }

  function stopResize(event: PointerEvent) {
    if (!isResizing.value) return;
    isResizing.value = false;
    try {
      (event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId);
    } catch {
      /* pointer already released */
    }
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(storageKey, String(Math.round(sidebarWidth.value)));
    }
  }

  return { containerRef, sidebarWidth, isResizing, startResize, onResize, stopResize };
}
