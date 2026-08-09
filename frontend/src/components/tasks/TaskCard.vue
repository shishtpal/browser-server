<template>
  <li
    class="rounded-xl border bg-white p-3.5 shadow-sm transition-colors sm:p-4 dark:bg-slate-800/90"
    :class="
      task.stale
        ? 'border-amber-300 dark:border-amber-700/60'
        : 'border-gray-200 dark:border-slate-700'
    "
  >
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0 flex-1">
        <!-- Badges -->
        <div class="mb-1.5 flex flex-wrap items-center gap-1.5">
          <span
            class="inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[10px] font-black tracking-wider uppercase"
            :class="meta.badgeClass"
          >
            <component :is="meta.icon" class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
            {{ meta.label }}
          </span>
          <span
            v-if="task.stale"
            class="inline-flex items-center gap-1 rounded-md bg-amber-100 px-2 py-0.5 text-[10px] font-black tracking-wider text-amber-700 uppercase dark:bg-amber-900/30 dark:text-amber-400"
            title="The worker's lease expired. The watchdog will requeue this shortly."
          >
            <TriangleAlert class="h-3 w-3" :stroke-width="2.5" aria-hidden="true" />
            Stale
          </span>
          <span
            v-if="task.has_checkpoint"
            class="inline-flex items-center gap-1 rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-black tracking-wider text-slate-600 uppercase dark:bg-slate-700 dark:text-slate-300"
            title="Resumable state exists — a retry resumes rather than restarts."
          >
            <BookmarkCheck class="h-3 w-3" :stroke-width="2.25" aria-hidden="true" />
            Checkpoint
          </span>
          <span
            v-if="task.attempts > 0"
            class="text-[10px] font-semibold text-slate-500 tabular-nums dark:text-slate-400"
          >
            attempt {{ task.attempts }}/{{ task.max_attempts }}
          </span>
        </div>

        <p
          class="text-sm font-medium break-words whitespace-pre-wrap text-slate-800 dark:text-slate-200"
        >
          {{ task.prompt }}
        </p>

        <p class="mt-1.5 text-[11px] text-slate-500 dark:text-slate-400">
          Created {{ timeAgo(task.created_at) }}
          <template v-if="task.completed_at"> · finished {{ timeAgo(task.completed_at) }}</template>
          <template v-else-if="task.last_progress">
            · last progress {{ timeAgo(task.last_progress) }}</template
          >
          <template v-else-if="queuedLater"> · runs {{ timeAgo(task.available_at) }}</template>
        </p>
      </div>

      <!-- Actions -->
      <div class="flex shrink-0 items-center gap-1">
        <button
          v-if="task.status === 'queued'"
          type="button"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-amber-50 hover:text-amber-600 dark:hover:bg-amber-900/20 dark:hover:text-amber-400"
          title="Cancel task"
          aria-label="Cancel task"
          @click="$emit('cancel', task.id)"
        >
          <Ban class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
        <button
          v-if="isTerminal"
          type="button"
          class="grid h-8 w-8 place-items-center rounded-lg text-slate-400 transition hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20 dark:hover:text-red-400"
          title="Delete task"
          aria-label="Delete task"
          @click="$emit('delete', task.id)"
        >
          <Trash2 class="h-4 w-4" :stroke-width="2.25" aria-hidden="true" />
        </button>
      </div>
    </div>

    <p
      v-if="task.last_error"
      class="mt-3 flex items-start gap-2 rounded-lg bg-red-50 px-3 py-2 text-xs font-semibold text-red-700 dark:bg-red-900/20 dark:text-red-400"
      role="alert"
    >
      <CircleAlert class="mt-0.5 h-3.5 w-3.5 shrink-0" :stroke-width="2.25" aria-hidden="true" />
      {{ task.last_error }}
    </p>

    <!-- Expandable result -->
    <div v-if="task.result" class="mt-3">
      <button
        type="button"
        class="inline-flex min-h-8 items-center gap-1.5 text-[11px] font-black tracking-wider text-violet-600 uppercase transition hover:text-violet-500 dark:text-violet-400 dark:hover:text-violet-300"
        :aria-expanded="expanded"
        @click="expanded = !expanded"
      >
        <ChevronDown
          class="h-3.5 w-3.5 transition-transform"
          :class="{ 'rotate-180': expanded }"
          :stroke-width="2.5"
          aria-hidden="true"
        />
        {{ expanded ? 'Hide' : 'Show' }} result · {{ task.result.steps }} step{{
          task.result.steps === 1 ? '' : 's'
        }}
      </button>
      <p
        v-if="expanded"
        class="mt-2 max-h-80 overflow-y-auto rounded-lg bg-gray-50 px-3 py-2 text-xs break-words whitespace-pre-wrap text-slate-700 dark:bg-slate-900 dark:text-slate-300"
      >
        {{ task.result.response }}
      </p>
    </div>
  </li>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { Ban, BookmarkCheck, ChevronDown, CircleAlert, Trash2, TriangleAlert } from '@lucide/vue';
import type { AITask } from '@browser-server/shared-types';
import { timeAgo } from '../../lib/utils';
import { TASK_STATUS_META } from './taskFormat';

const props = defineProps<{ task: AITask }>();

defineEmits<{
  cancel: [id: string];
  delete: [id: string];
}>();

const expanded = ref(false);

const meta = computed(() => TASK_STATUS_META[props.task.status]);

const isTerminal = computed(
  () => props.task.status === 'completed' || props.task.status === 'failed',
);

/** A retry backs a task off, so its next run is in the future. */
const queuedLater = computed(
  () => props.task.status === 'queued' && new Date(props.task.available_at).getTime() > Date.now(),
);
</script>
