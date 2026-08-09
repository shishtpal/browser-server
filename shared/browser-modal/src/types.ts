/**
 * Shared types for the imperative modal service.
 */

/** Visual intent of a dialog — drives the icon and the accent color. */
export type ModalKind = 'info' | 'confirm' | 'danger';

export interface ModalOptions {
  /** Label of the cancel/dismiss button (confirm dialogs only). */
  cancelText?: string;
  /** Label of the primary button. */
  confirmText?: string;
  /** When true, clicking the backdrop or pressing Escape does not dismiss. */
  persistent?: boolean;
  /** Extra class for the dialog panel (escape hatch for callers). */
  panelClass?: string;
}

/** A queued, unresolved request rendered by <ModalHost />. */
export interface ModalRequest {
  id: number;
  kind: ModalKind;
  title: string;
  message?: string;
  cancelText: string;
  confirmText: string;
  persistent: boolean;
  panelClass?: string;
  /**
   * true  → confirmed
   * false → cancelled (cancel button / backdrop / escape)
   * Also resolves (false) if programmatically dismissed while buried in queue.
   */
  resolve: (value: boolean) => void;
}

/** Public imperative API surface. */
export interface ModalApi {
  /** Info/confirm dialog, resolves true/false. */
  confirm: (title: string, message?: string, options?: ModalOptions) => Promise<boolean>;
  /** Destructive confirm (red): "Delete permanently?" style. */
  confirmDelete: (title: string, message?: string, options?: ModalOptions) => Promise<boolean>;
  /** Dismissable notice with a single OK button; resolves when closed. */
  alert: (title: string, message?: string, options?: ModalOptions) => Promise<void>;
  /** Number of dialogs currently queued (debugging/tests). */
  pendingCount: () => number;
}
