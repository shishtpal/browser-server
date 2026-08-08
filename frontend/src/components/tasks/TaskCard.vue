<template>
  <li
    class="rounded-xl border bg-white p-4 shadow-sm transition-colors dark:bg-slate-800/90"
    :class="
      stale ? 'border-amber-300 dark:border-amber-700/60' : 'border-gray-200 dark:border-slate-700'
    "
  >
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0 flex-1">
        <div class="mb-1.5 flex flex-wrap items-center gap-1.5">
          <span
            class="rounded-md px-2 py-0.5 text-[10px] font-black tracking-wider uppercase"
            :class="statusClass"
          >
            {{ task.status }}
          </span>
          <span
            v-if="stale"
            class="rounded-md bg-amber-100 px-2 py-0.5 text-[10px] font-black tracking-wider text-amber-700 uppercase dark:bg-amber-900/30 dark:text-amber-400"
            title="The worker's lease expired. The watchdog will requeue this shortly."
          >
            Stale
          </span>
          <span
            v-if="task.has_checkpoint"
            class="rounded-md bg-gray-100 px-2 py-0.5 text-[10px] font-black tracking-wider text-slate-600 uppercase dark:bg-slate-700 dark:text-slate-300"
            title="Resumable state exists — a retry resumes rather than restarts."
          >
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

      <div class="flex shrink-0 items-center gap-2">
        <Button
          v-if="task.status === 'queued'"
          variant="ghost"
          size="sm"
          @click="emit('cancel', task.id)"
          >Cancel</Button
        >
        <Button v-if="isTerminal" variant="danger" size="sm" @click="emit('delete', task.id)"
          >Delete</Button
        >
      </div>
    </div>

    <p
      v-if="task.last_error"
      class="mt-3 rounded-lg bg-red-50 px-3 py-2 text-xs font-semibold text-red-700 dark:bg-red-900/20 dark:text-red-400"
    >
      {{ task.last_error }}
    </p>

    <div v-if="task.result" class="mt-3">
      <button
        type="button"
        class="text-[11px] font-black tracking-wider text-violet-600 uppercase transition hover:text-violet-500 dark:text-violet-400"
        @click="expanded = !expanded"
      >
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
import type { AITask } from '@browser-server/shared-types'
import { computed, ref } from 'vue'
import Button from '../ui/Button.vue'
import { timeAgo } from '../../lib/utils'

const props = defineProps<{ task: AITask }>()

const emit = defineEmits<{
  cancel: [id: string]
  delete: [id: string]
}>()

const expanded = ref(false)

const stale = computed(() => props.task.stale)
const isTerminal = computed(
  () => props.task.status === 'completed' || props.task.status === 'failed',
)

/** A retry backs a task off, so its next run is in the future. */
const queuedLater = computed(
  () => props.task.status === 'queued' && new Date(props.task.available_at).getTime() > Date.now(),
)

const statusClass = computed(() => {
  const classes: Record<AITask['status'], string> = {
    queued: 'bg-gray-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300',
    running: 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400',
    completed: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
    failed: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  }
  return classes[props.task.status]
})
</script>
