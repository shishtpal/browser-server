<template>
  <Teleport to="body" :disabled="!canTeleport">
    <Transition name="bsm-backdrop">
      <div
        v-if="active"
        class="bsm-backdrop"
        role="presentation"
        @click.self="onBackdropClick"
        @keydown="onKeydown"
      >
        <Transition name="bsm-dialog" appear>
          <ConfirmDialog
            :key="active.id"
            ref="dialog"
            :request="active"
            @confirm="onConfirm"
            @cancel="onCancel"
          />
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import ConfirmDialog from './ConfirmDialog.vue';
import { activeModal, settleModal } from './store';

const active = activeModal;
const dialog = ref<InstanceType<typeof ConfirmDialog> | null>(null);

/** Teleport is only available client-side (Astro SSR safety). */
const canTeleport = ref(false);
onMounted(() => (canTeleport.value = true));

function onConfirm() {
  if (active.value) settleModal(active.value.id, true);
}

function onCancel() {
  if (active.value) settleModal(active.value.id, false);
}

function onBackdropClick() {
  if (active.value && !active.value.persistent) onCancel();
}

function onKeydown(event: KeyboardEvent) {
  if (!active.value) return;
  if (event.key === 'Escape' && !active.value.persistent) {
    event.stopPropagation();
    onCancel();
    return;
  }
  dialog.value?.trapFocus(event);
}

/* --------------------- body scroll lock while a dialog is open --------------------- */

let previousOverflow: string | null = null;
let previousFocused: HTMLElement | null = null;

watch(active, (current, previous) => {
  if (current && !previous) {
    previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    if (document.activeElement instanceof HTMLElement) {
      previousFocused = document.activeElement;
    }
  } else if (!current && previous) {
    document.body.style.overflow = previousOverflow ?? '';
    previousOverflow = null;
    previousFocused?.focus?.();
    previousFocused = null;
  }
});

onBeforeUnmount(() => {
  if (previousOverflow !== null) document.body.style.overflow = previousOverflow;
});

/** Escape key works even when focus is somehow outside the dialog. */
const onGlobalKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && active.value && !active.value.persistent) onCancel();
};
onMounted(() => window.addEventListener('keydown', onGlobalKeydown));
onBeforeUnmount(() => window.removeEventListener('keydown', onGlobalKeydown));
</script>

<style>
@import './modal.css';
</style>
