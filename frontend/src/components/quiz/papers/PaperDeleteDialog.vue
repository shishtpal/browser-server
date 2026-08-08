<template>
  <Modal :open="!!paper" title="Delete question paper?" @close="$emit('cancel')">
    <div class="flex items-start gap-3">
      <span
        class="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-rose-100 text-rose-600 dark:bg-rose-900/30 dark:text-rose-400"
      >
        <TriangleAlert class="h-5 w-5" :stroke-width="2.25" aria-hidden="true" />
      </span>
      <p class="text-xs leading-relaxed text-slate-500 dark:text-slate-400">
        Are you sure you want to delete
        <span class="font-bold text-slate-800 dark:text-slate-200">"{{ paper?.title }}"</span>? This
        action cannot be undone.
      </p>
    </div>

    <div class="mt-6 flex justify-end gap-2">
      <Button variant="secondary" size="sm" @click="$emit('cancel')">Cancel</Button>
      <Button variant="danger" size="sm" @click="$emit('confirm')">
        <span class="inline-flex items-center gap-1.5">
          <Trash2 class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
          Delete
        </span>
      </Button>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { Trash2, TriangleAlert } from '@lucide/vue';
import type { QuestionPaper } from '../../../types';
import Button from '../../ui/Button.vue';
import Modal from '../../ui/Modal.vue';

defineProps<{ paper: QuestionPaper | null }>();
defineEmits<{ confirm: []; cancel: [] }>();
</script>
