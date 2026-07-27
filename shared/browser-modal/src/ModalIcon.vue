<script setup lang="ts">
import { computed } from 'vue'
import type { ModalIconType } from './types'

const props = defineProps<{ icon: ModalIconType; small?: boolean }>()

const color = computed(() => ({
  success: 'text-emerald-500',
  error: 'text-red-500',
  warning: 'text-amber-500',
  info: 'text-sky-500',
  question: 'text-violet-500',
}[props.icon]))
</script>

<template>
  <svg
    viewBox="0 0 52 52"
    fill="none"
    stroke="currentColor"
    stroke-width="2.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    :class="[color, small ? 'size-7 shrink-0' : 'mx-auto size-20']"
  >
    <circle class="swal-ico-circle" cx="26" cy="26" r="23" opacity="0.45" />

    <template v-if="icon === 'success'">
      <path class="swal-ico-draw" d="M15.5 27 22.5 34 37 19" />
    </template>

    <template v-else-if="icon === 'error'">
      <path class="swal-ico-draw" style="--swal-d: .18s" d="M18.5 18.5 33.5 33.5" />
      <path class="swal-ico-draw" style="--swal-d: .30s" d="M33.5 18.5 18.5 33.5" />
    </template>

    <template v-else-if="icon === 'warning'">
      <path class="swal-ico-draw" d="M26 14.5v16" />
      <circle class="swal-ico-dot" cx="26" cy="37" r="1.9" fill="currentColor" stroke="none" />
    </template>

    <template v-else-if="icon === 'info'">
      <path class="swal-ico-draw" d="M26 24v14" />
      <circle class="swal-ico-dot" cx="26" cy="16" r="1.9" fill="currentColor" stroke="none" />
    </template>

    <template v-else>
      <path class="swal-ico-draw" d="M19.8 20.6a6.4 6.4 0 1 1 6.2 7.4v3.5" />
      <circle class="swal-ico-dot" cx="26" cy="37" r="1.9" fill="currentColor" stroke="none" />
    </template>
  </svg>
</template>
