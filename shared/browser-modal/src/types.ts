export type ModalIconType = 'success' | 'error' | 'warning' | 'info' | 'question'

export type ModalPosition =
  | 'center' | 'top' | 'top-start' | 'top-end'
  | 'bottom' | 'bottom-start' | 'bottom-end'

export type DismissReason = 'backdrop' | 'esc' | 'close' | 'cancel' | 'timer'

export type ModalInputType =
  | 'text' | 'email' | 'password' | 'number' | 'tel' | 'url'
  | 'textarea' | 'select' | 'checkbox'

export type ButtonVariant = 'primary' | 'danger' | 'success' | 'warning' | 'neutral'

export interface ModalOptions<T = unknown> {
  title?: string
  text?: string
  /** Raw HTML — sanitize before passing user content. */
  html?: string
  footer?: string
  icon?: ModalIconType

  /* buttons */
  showConfirmButton?: boolean
  showCancelButton?: boolean
  showDenyButton?: boolean
  showCloseButton?: boolean
  confirmButtonText?: string
  cancelButtonText?: string
  denyButtonText?: string
  confirmButtonVariant?: ButtonVariant
  denyButtonVariant?: ButtonVariant
  reverseButtons?: boolean
  focusCancel?: boolean

  /* behaviour */
  allowOutsideClick?: boolean
  allowEscapeKey?: boolean
  timer?: number
  timerProgressBar?: boolean

  /* layout */
  position?: ModalPosition
  toast?: boolean
  width?: string

  /* input */
  input?: ModalInputType
  inputLabel?: string
  inputPlaceholder?: string
  inputValue?: string | number | boolean
  inputAttributes?: Record<string, string | number>
  /** Return an error string to block confirmation. */
  inputValidator?: (value: unknown) => string | null | undefined | Promise<string | null | undefined>

  /** Return `false` to abort. Any other returned value becomes `result.value`. */
  preConfirm?: (value: unknown) => T | false | Promise<T | false>

  /** Extra classes merged into the panel. */
  customClass?: string
}

export interface ModalResult<T = unknown> {
  isConfirmed: boolean
  isDenied: boolean
  isDismissed: boolean
  value?: T
  dismiss?: DismissReason
}

/** A live modal on the stack. `pendingClose` is set externally to trigger an animated close. */
export interface ModalInstance {
  id: number
  options: ModalOptions
  pendingClose: ModalResult | null
  resolve: (r: ModalResult) => void
}
