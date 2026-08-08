<template>
  <Modal :open="open" title="Submit exam?" @close="$emit('cancel')">
    <div class="space-y-3">
      <p class="text-xs leading-relaxed text-slate-500 dark:text-slate-400">
        You have answered
        <span class="font-bold text-violet-600 tabular-nums">{{ answered }}</span> out of
        <span class="font-bold tabular-nums">{{ total }}</span> questions.
      </p>

      <p
        v-if="total - answered > 0"
        class="flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-700 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-300"
        role="alert"
      >
        <TriangleAlert class="h-4 w-4 shrink-0" :stroke-width="2.25" aria-hidden="true" />
        {{ total - answered }} question{{ total - answered === 1 ? '' : 's' }} remain{{
          total - answered === 1 ? 's' : ''
        }}
        unanswered!
      </p>

      <div class="flex flex-col-reverse gap-2 pt-2 sm:flex-row sm:justify-end">
        <Button variant="secondary" size="sm" class="w-full sm:w-auto" @click="$emit('cancel')">
          Continue exam
        </Button>
        <Button
          variant="gradient-emerald"
          size="sm"
          class="w-full sm:w-auto"
          @click="$emit('confirm')"
        >
          <span class="inline-flex items-center gap-1.5">
            <Send class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
            Submit now
          </span>
        </Button>
      </div>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { Send, TriangleAlert } from '@lucide/vue';
import Button from '../../../ui/Button.vue';
import Modal from '../../../ui/Modal.vue';

defineProps<{ open: boolean; answered: number; total: number }>();
defineEmits<{ confirm: []; cancel: [] }>();
</script>
