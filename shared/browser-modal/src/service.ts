import { ref, computed, reactive } from 'vue'
import type { ModalIconType, ModalInstance, ModalOptions, ModalResult, DismissReason } from './types'

let seed = 0

export const stack = ref<ModalInstance[]>([])
export const isVisible = computed(() => stack.value.length > 0)

const DEFAULTS: ModalOptions = {
  showConfirmButton: true,
  showCancelButton: false,
  showDenyButton: false,
  showCloseButton: false,
  confirmButtonText: 'OK',
  cancelButtonText: 'Cancel',
  denyButtonText: 'No',
  allowOutsideClick: true,
  allowEscapeKey: true,
  position: 'center',
  toast: false,
  timerProgressBar: true,
}

function normalize(
  a: ModalOptions | string,
  b?: string,
  c?: ModalIconType,
): ModalOptions {
  const raw = typeof a === 'string' ? { title: a, text: b, icon: c } : a
  return { ...DEFAULTS, ...raw }
}

/** Core API — mirrors Swal.fire() signatures. */
export function fire<T = unknown>(options: ModalOptions<T>): Promise<ModalResult<T>>
export function fire<T = unknown>(title: string, text?: string, icon?: ModalIconType): Promise<ModalResult<T>>
export function fire<T = unknown>(a: ModalOptions | string, b?: string, c?: ModalIconType): Promise<ModalResult<T>> {
  const options = normalize(a, b, c)
  return new Promise<ModalResult<T>>((resolve) => {
    const instance = reactive<ModalInstance>({
      id: ++seed,
      options,
      pendingClose: null,
      resolve: resolve as (r: ModalResult) => void,
    })
    stack.value.push(instance)
  })
}

/** Called by ModalDialog after its leave animation completes. */
export function destroy(id: number, result: ModalResult) {
  const i = stack.value.findIndex(m => m.id === id)
  if (i === -1) return
  const [inst] = stack.value.splice(i, 1)
  inst.resolve(result)
}

function dismissed(reason: DismissReason = 'close'): ModalResult {
  return { isConfirmed: false, isDenied: false, isDismissed: true, dismiss: reason }
}

/** Programmatically close the top-most modal. */
export function close(result: ModalResult = dismissed()) {
  const top = stack.value.at(-1)
  if (top) top.pendingClose = result
}

/** Close all open modals. */
export function closeAll(result: ModalResult = dismissed()) {
  stack.value.forEach(m => (m.pendingClose = result))
}

/* ─── Sugar shortcuts ─────────────────────────────────────────────── */

const preset = (icon: ModalIconType) =>
  <T = unknown>(title: string, text?: string, extra: ModalOptions<T> = {}) =>
    fire<T>({ icon, title, text, ...extra })

export const modal = {
  fire,
  close,
  closeAll,
  isVisible,

  success: preset('success'),
  error: preset('error'),
  warning: preset('warning'),
  info: preset('info'),
  question: preset('question'),

  /** Confirm dialog — resolves boolean. */
  async confirm(title: string, text?: string, extra: ModalOptions = {}): Promise<boolean> {
    const r = await fire({
      icon: 'question',
      title,
      text,
      showCancelButton: true,
      confirmButtonText: 'Yes',
      cancelButtonText: 'Cancel',
      ...extra,
    })
    return r.isConfirmed
  },

  /** Destructive confirm (delete). */
  async confirmDelete(title = 'Delete this item?', text = "You won't be able to revert this!") {
    const r = await fire({
      icon: 'warning',
      title,
      text,
      showCancelButton: true,
      confirmButtonText: 'Yes, delete it',
      confirmButtonVariant: 'danger',
      reverseButtons: true,
      focusCancel: true,
    })
    return r.isConfirmed
  },

  /** Prompt for a value. */
  prompt<T = string>(title: string, opts: ModalOptions<T> = {}) {
    return fire<T>({
      title,
      input: 'text',
      showCancelButton: true,
      ...opts,
    })
  },

  /** Toast notification. */
  toast(title: string, icon: ModalIconType = 'success', timer = 3000) {
    return fire({
      toast: true,
      position: 'top-end',
      title,
      icon,
      timer,
      timerProgressBar: true,
      showConfirmButton: false,
      allowOutsideClick: false,
    })
  },
}

export function useModal() {
  return modal
}
