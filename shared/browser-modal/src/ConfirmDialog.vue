<template>
  <div
    ref="panel"
    class="bsm-panel"
    :class="[request.panelClass, `bsm-panel--${request.kind}`]"
    role="alertdialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    :aria-describedby="request.message ? messageId : undefined"
  >
    <div class="bsm-body">
      <div class="bsm-icon" aria-hidden="true">
        <component :is="kindIcon" class="bsm-icon-svg" :stroke-width="2.25" />
      </div>
      <div class="bsm-text">
        <h2 :id="titleId" class="bsm-title">{{ request.title }}</h2>
        <p v-if="request.message" :id="messageId" class="bsm-message">{{ request.message }}</p>
      </div>
    </div>

    <div class="bsm-actions">
      <button
        v-if="request.kind !== 'info'"
        ref="cancelButton"
        type="button"
        class="bsm-button bsm-button--ghost"
        @click="$emit('cancel')"
      >
        {{ request.cancelText }}
      </button>
      <button
        ref="confirmButton"
        type="button"
        class="bsm-button"
        :class="request.kind === 'danger' ? 'bsm-button--danger' : 'bsm-button--primary'"
        @click="$emit('confirm')"
      >
        {{ request.confirmText }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { OctagonX, CircleHelp, Info, type LucideIcon } from '@lucide/vue';
import type { ModalRequest } from './types';

const props = defineProps<{ request: ModalRequest }>();
defineEmits<{ confirm: []; cancel: [] }>();

const titleId = `bsm-title-${props.request.id}`;
const messageId = `bsm-message-${props.request.id}`;

const icons: Record<ModalRequest['kind'], LucideIcon> = {
  info: Info,
  confirm: CircleHelp,
  danger: OctagonX,
};
const kindIcon = computed(() => icons[props.request.kind]);

const panel = ref<HTMLElement | null>(null);
const confirmButton = ref<HTMLButtonElement | null>(null);
const cancelButton = ref<HTMLButtonElement | null>(null);

/** Focus the primary action on open (the host remounts this per request). */
onMounted(() => {
  (confirmButton.value ?? cancelButton.value)?.focus();
});

/** Focus trap: keep Tab/Shift+Tab cycling inside the panel. */
defineExpose({
  trapFocus: (event: KeyboardEvent) => {
    if (event.key !== 'Tab' || !panel.value) return;
    const focusables = Array.from(
      panel.value.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
      ),
    ).filter((el) => !el.hasAttribute('disabled'));
    if (!focusables.length) return;
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    const active = document.activeElement as HTMLElement | null;
    if (event.shiftKey && active === first) {
      last.focus();
      event.preventDefault();
    } else if (!event.shiftKey && active === last) {
      first.focus();
      event.preventDefault();
    }
  },
});
</script>
