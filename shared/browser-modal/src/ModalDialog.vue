<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import ModalIcon from './ModalIcon.vue'
import { destroy } from './service'
import type { ButtonVariant, DismissReason, ModalInstance, ModalResult } from './types'

const props = defineProps<{ instance: ModalInstance; isTop: boolean }>()

const o = computed(() => props.instance.options)

const open = ref(false)
const closing = ref(false)
const loading = ref(false)
const panel = ref<HTMLElement | null>(null)
const inputEl = ref<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement | null>(null)
// Polymorphic: boolean for checkbox, string/number for text/select — no single
// non-any type satisfies all v-model bindings; the value is read as unknown downstream.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const inputValue = ref<any>(o.value.inputValue ?? (o.value.input === 'checkbox' ? false : ''))
const inputError = ref<string | null>(null)
const progress = ref(100)

let result: ModalResult | null = null
let lastFocused: HTMLElement | null = null
let raf = 0

/* ─── Layout ─────────────────────────────────────────────────────── */

const positionClass = computed(() => ({
  'center': 'items-center justify-center',
  'top': 'items-start justify-center',
  'top-start': 'items-start justify-start',
  'top-end': 'items-start justify-end',
  'bottom': 'items-end justify-center',
  'bottom-start': 'items-end justify-start',
  'bottom-end': 'items-end justify-end',
}[o.value.position ?? 'center']))

const autoVariant = computed<ButtonVariant>(() => ({
  success: 'success',
  error: 'danger',
  warning: 'warning',
  info: 'primary',
  question: 'primary',
}[o.value.icon ?? 'info'] as ButtonVariant))

const VARIANTS: Record<ButtonVariant, string> = {
  primary: 'bg-sky-600 text-white hover:bg-sky-500 focus-visible:ring-sky-500',
  danger: 'bg-red-600 text-white hover:bg-red-500 focus-visible:ring-red-500',
  success: 'bg-emerald-600 text-white hover:bg-emerald-500 focus-visible:ring-emerald-500',
  warning: 'bg-amber-500 text-white hover:bg-amber-400 focus-visible:ring-amber-500',
  neutral: 'bg-slate-200 text-slate-800 hover:bg-slate-300 focus-visible:ring-slate-400 dark:bg-slate-700 dark:text-slate-100 dark:hover:bg-slate-600',
}

const btnBase =
  'inline-flex min-w-24 items-center justify-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold ' +
  'transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 ' +
  'focus-visible:ring-offset-white dark:focus-visible:ring-offset-slate-800 ' +
  'disabled:cursor-not-allowed disabled:opacity-60 cursor-pointer'

const confirmClass = computed(() => `${btnBase} ${VARIANTS[o.value.confirmButtonVariant ?? autoVariant.value]}`)
const denyClass = computed(() => `${btnBase} ${VARIANTS[o.value.denyButtonVariant ?? 'danger']}`)
const cancelClass = computed(() => `${btnBase} ${VARIANTS.neutral}`)

/* ─── Close flow ─────────────────────────────────────────────────── */

function finish(r: ModalResult) {
  if (closing.value) return
  closing.value = true
  result = r
  open.value = false
}

function dismiss(reason: DismissReason) {
  finish({ isConfirmed: false, isDenied: false, isDismissed: true, dismiss: reason })
}

function onAfterLeave() {
  destroy(props.instance.id, result ?? { isConfirmed: false, isDenied: false, isDismissed: true, dismiss: 'close' })
}

watch(() => props.instance.pendingClose, v => { if (v) finish(v) })

async function onConfirm() {
  if (loading.value) return
  inputError.value = null
  let value: unknown = undefined

  if (o.value.input) {
    value = o.value.input === 'checkbox' ? !!inputValue.value : inputValue.value
    if (o.value.inputValidator) {
      loading.value = true
      const err = await o.value.inputValidator(value)
      loading.value = false
      if (err) { inputError.value = err; inputEl.value?.focus(); return }
    }
  }

  if (o.value.preConfirm) {
    loading.value = true
    try {
      const r = await o.value.preConfirm(value)
      if (r === false) { loading.value = false; return }
      value = r
    } catch (e: unknown) {
      inputError.value = e instanceof Error ? e.message : String(e)
      loading.value = false
      return
    }
    loading.value = false
  }

  finish({ isConfirmed: true, isDenied: false, isDismissed: false, value })
}

function onDeny() {
  finish({ isConfirmed: false, isDenied: true, isDismissed: false })
}

function onBackdrop() {
  if (o.value.allowOutsideClick && !loading.value) dismiss('backdrop')
}

/* ─── A11y: escape + focus trap ──────────────────────────────────── */

const FOCUSABLE =
  'a[href],button:not([disabled]),textarea:not([disabled]),input:not([disabled]):not([type="hidden"]),select:not([disabled]),[tabindex]:not([tabindex="-1"])'

function onKeydown(e: KeyboardEvent) {
  if (!props.isTop || closing.value) return

  if (e.key === 'Escape' && o.value.allowEscapeKey && !loading.value) {
    e.preventDefault()
    dismiss('esc')
    return
  }

  if (e.key === 'Tab' && panel.value) {
    const nodes = Array.from(panel.value.querySelectorAll<HTMLElement>(FOCUSABLE))
      .filter(el => el.offsetParent !== null)
    if (!nodes.length) return
    const first = nodes[0]
    const last = nodes[nodes.length - 1]
    const active = document.activeElement as HTMLElement

    if (e.shiftKey && (active === first || !panel.value.contains(active))) {
      e.preventDefault(); last.focus()
    } else if (!e.shiftKey && active === last) {
      e.preventDefault(); first.focus()
    }
  }
}

/* ─── Scroll lock ────────────────────────────────────────────────── */

let locked = false

function lockScroll() {
  const w = window as unknown as Record<string, number>
  if (w.__swalLocks == null) w.__swalLocks = 0
  if (w.__swalLocks++ === 0) {
    const gap = window.innerWidth - document.documentElement.clientWidth
    document.body.style.overflow = 'hidden'
    if (gap > 0) document.body.style.paddingRight = `${gap}px`
  }
  locked = true
}

function unlockScroll() {
  if (!locked) return
  const w = window as unknown as Record<string, number>
  if (--w.__swalLocks <= 0) {
    document.body.style.overflow = ''
    document.body.style.paddingRight = ''
  }
}

/* ─── Timer ──────────────────────────────────────────────────────── */

function startTimer(ms: number) {
  const t0 = performance.now()
  const tick = (t: number) => {
    const elapsed = t - t0
    progress.value = Math.max(0, 100 - (elapsed / ms) * 100)
    if (elapsed >= ms) dismiss('timer')
    else raf = requestAnimationFrame(tick)
  }
  raf = requestAnimationFrame(tick)
}

/* ─── Lifecycle ──────────────────────────────────────────────────── */

onMounted(async () => {
  lastFocused = document.activeElement as HTMLElement
  if (!o.value.toast) lockScroll()
  document.addEventListener('keydown', onKeydown, true)
  open.value = true

  await nextTick()
  if (o.value.input) inputEl.value?.focus()
  else if (o.value.focusCancel) panel.value?.querySelector<HTMLElement>('[data-cancel]')?.focus()
  else panel.value?.querySelector<HTMLElement>('[data-confirm]')?.focus()
    ?? panel.value?.focus()

  if (o.value.timer) startTimer(o.value.timer)
})

onBeforeUnmount(() => {
  cancelAnimationFrame(raf)
  document.removeEventListener('keydown', onKeydown, true)
  unlockScroll()
  lastFocused?.focus?.()
})
</script>

<template>
  <Transition name="swal" :duration="{ enter: 320, leave: 200 }" @after-leave="onAfterLeave">
    <div
      v-if="open"
      class="swal-root fixed inset-0 z-[9999] flex overflow-y-auto"
      :class="[positionClass, o.toast ? 'pointer-events-none p-4' : 'p-4 sm:p-6']"
    >
      <!-- backdrop -->
      <div
        v-if="!o.toast"
        class="swal-backdrop fixed inset-0 bg-slate-900/60 backdrop-blur-[2px]"
        @mousedown.self="onBackdrop"
      />

      <!-- panel -->
      <div
        ref="panel"
        class="swal-panel relative my-auto w-full"
        :class="[
          o.toast
            ? 'pointer-events-auto max-w-sm rounded-xl bg-white p-4 shadow-xl ring-1 ring-slate-900/5 dark:bg-slate-800 dark:ring-white/10'
            : 'max-w-md rounded-2xl bg-white p-6 text-center shadow-2xl ring-1 ring-slate-900/5 sm:p-8 dark:bg-slate-800 dark:ring-white/10',
          o.customClass,
        ]"
        :style="o.width ? { maxWidth: o.width } : undefined"
        role="dialog"
        aria-modal="true"
        :aria-label="o.title || 'Dialog'"
        tabindex="-1"
      >
        <!-- close button -->
        <button
          v-if="o.showCloseButton"
          type="button"
          aria-label="Close"
          class="absolute right-3 top-3 grid size-8 place-items-center rounded-full text-slate-400
                 transition hover:bg-slate-100 hover:text-slate-600 focus:outline-none
                 focus-visible:ring-2 focus-visible:ring-sky-500 dark:hover:bg-slate-700 dark:hover:text-slate-200"
          @click="dismiss('close')"
        >
          <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" class="size-5">
            <path d="M5 5l10 10M15 5L5 15" stroke-linecap="round" />
          </svg>
        </button>

        <!-- TOAST layout -->
        <template v-if="o.toast">
          <div class="flex items-center gap-3 pr-2">
            <ModalIcon v-if="o.icon" :icon="o.icon" small />
            <div class="min-w-0 flex-1 text-left">
              <p v-if="o.title" class="truncate text-sm font-semibold text-slate-900 dark:text-slate-50">{{ o.title }}</p>
              <p v-if="o.text" class="mt-0.5 text-xs text-slate-500 dark:text-slate-400">{{ o.text }}</p>
            </div>
          </div>
        </template>

        <!-- DIALOG layout -->
        <template v-else>
          <ModalIcon v-if="o.icon" :icon="o.icon" class="mb-5" />

          <h2 v-if="o.title" class="text-xl font-bold text-slate-900 dark:text-slate-50">
            {{ o.title }}
          </h2>

          <p v-if="o.text" class="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">
            {{ o.text }}
          </p>

          <!-- eslint-disable-next-line vue/no-v-html -->
          <div v-if="o.html" class="swal-html mt-2 text-sm text-slate-600 dark:text-slate-300" v-html="o.html" />

          <!-- input -->
          <div v-if="o.input" class="mt-5 text-left">
            <label v-if="o.inputLabel" class="mb-1.5 block text-sm font-medium text-slate-700 dark:text-slate-200">
              {{ o.inputLabel }}
            </label>

            <textarea
              v-if="o.input === 'textarea'"
              ref="inputEl"
              v-model="inputValue"
              rows="3"
              :placeholder="o.inputPlaceholder"
              class="w-full resize-y rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900
                     placeholder:text-slate-400 focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500/30
                     dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
              @keydown.enter.meta="onConfirm"
            />

            <select
              v-else-if="o.input === 'select'"
              ref="inputEl"
              v-model="inputValue"
              class="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900
                     focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500/30
                     dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
            />

            <label v-else-if="o.input === 'checkbox'" class="flex items-center gap-2.5 text-sm text-slate-700 dark:text-slate-200">
              <input
                ref="inputEl"
                v-model="inputValue"
                type="checkbox"
                class="size-4 rounded border-slate-300 text-sky-600 focus:ring-sky-500 dark:border-slate-600 dark:bg-slate-900"
              >
              <span>{{ o.inputPlaceholder }}</span>
            </label>

            <input
              v-else
              ref="inputEl"
              v-model="inputValue"
              :type="o.input"
              :placeholder="o.inputPlaceholder"
              class="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900
                     placeholder:text-slate-400 focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500/30
                     dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
              @keydown.enter.prevent="onConfirm"
            >
          </div>

          <p v-if="inputError" class="mt-2 flex items-center gap-1.5 rounded-md bg-red-50 px-3 py-2 text-left text-xs font-medium text-red-600 dark:bg-red-500/10 dark:text-red-400">
            <svg viewBox="0 0 20 20" fill="currentColor" class="size-4 shrink-0">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9 5h2v6H9V5zm0 8h2v2H9v-2z" clip-rule="evenodd" />
            </svg>
            {{ inputError }}
          </p>

          <!-- actions -->
          <div
            v-if="o.showConfirmButton || o.showCancelButton || o.showDenyButton"
            class="mt-7 flex flex-wrap justify-center gap-3"
            :class="o.reverseButtons && 'flex-row-reverse'"
          >
            <button v-if="o.showConfirmButton" data-confirm type="button" :class="confirmClass" :disabled="loading" @click="onConfirm">
              <svg v-if="loading" class="size-4 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" class="opacity-25" />
                <path d="M12 2a10 10 0 0110 10" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
              </svg>
              {{ o.confirmButtonText }}
            </button>

            <button v-if="o.showDenyButton" type="button" :class="denyClass" :disabled="loading" @click="onDeny">
              {{ o.denyButtonText }}
            </button>

            <button v-if="o.showCancelButton" data-cancel type="button" :class="cancelClass" :disabled="loading" @click="dismiss('cancel')">
              {{ o.cancelButtonText }}
            </button>
          </div>

          <p v-if="o.footer" class="mt-6 border-t border-slate-200 pt-4 text-xs text-slate-500 dark:border-slate-700 dark:text-slate-400">
            {{ o.footer }}
          </p>
        </template>

        <!-- timer bar -->
        <div
          v-if="o.timer && o.timerProgressBar"
          class="absolute inset-x-0 bottom-0 h-1 overflow-hidden rounded-b-2xl bg-slate-200/70 dark:bg-slate-700/70"
        >
          <div class="h-full bg-sky-500 transition-none" :style="{ width: progress + '%' }" />
        </div>
      </div>
    </div>
  </Transition>
</template>
