<template>
  <div class="mx-auto max-w-full px-4 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Background agent" title="Tasks" color="violet">
      <template #stats>
        <StatCard :value="counts.running ?? 0" label="Running" variant="primary" color="violet" />
        <StatCard :value="counts.queued ?? 0" label="Queued" variant="dark" color="violet" />
      </template>
      <template #actions>
        <Button variant="secondary" size="sm" :disabled="isLoading" @click="load()">Refresh</Button>
      </template>
    </PageHeader>

    <ErrorBanner v-if="error" :message="error" :on-retry="() => load()" />

    <LoadingSpinner v-if="isLoading && !tasks.length" message="Loading tasks..." color="violet" />

    <!-- Runner disabled: explain the fix rather than showing an empty queue -->
    <EmptyState
      v-else-if="status && !enabled"
      title="Background tasks are disabled"
      description="Set tasks.enabled to true in bs-ai-config.json and restart the server to run unattended agent work."
      icon="clock"
      color="violet"
    />

    <template v-else>
      <!-- Submit -->
      <form
        class="mb-4 rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition-colors dark:border-slate-700 dark:bg-slate-800/90"
        @submit.prevent="handleSubmit"
      >
        <label
          for="task-prompt"
          class="mb-2 block text-xs font-black tracking-wider text-slate-500 uppercase"
        >
          New task
        </label>
        <textarea
          id="task-prompt"
          v-model="draft"
          rows="3"
          :disabled="isSubmitting"
          placeholder="Describe what the agent should do. It runs unattended, so avoid anything that needs your input."
          class="w-full resize-y rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-slate-800 transition outline-none focus:border-violet-400 disabled:opacity-60 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200"
          @keydown.ctrl.enter.prevent="handleSubmit"
          @keydown.meta.enter.prevent="handleSubmit"
        />
        <div class="mt-2 flex items-center justify-between gap-3">
          <p class="text-[11px] text-slate-500 dark:text-slate-400">
            {{ workers }} worker{{ workers === 1 ? '' : 's' }} · Ctrl+Enter to submit
          </p>
          <Button
            type="submit"
            variant="gradient-violet"
            size="sm"
            :loading="isSubmitting"
            loading-text="Queueing..."
            :disabled="!draft.trim()"
          >
            Queue task
          </Button>
        </div>
      </form>

      <!-- Status filter -->
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <button
          v-for="option in filterOptions"
          :key="option.value"
          type="button"
          class="rounded-lg px-3 py-1.5 text-xs font-semibold transition"
          :class="
            filter === option.value
              ? 'bg-violet-100 text-violet-700 ring-1 ring-violet-300 dark:bg-violet-900/30 dark:text-violet-300 dark:ring-violet-700'
              : 'bg-gray-100 text-slate-600 hover:bg-gray-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
          "
          @click="setFilter(option.value)"
        >
          {{ option.label }}
          <span v-if="countFor(option.value) !== null" class="ml-1 tabular-nums opacity-60">{{
            countFor(option.value)
          }}</span>
        </button>
        <span
          v-if="hasActive"
          class="ml-auto flex items-center gap-1.5 text-[11px] font-medium text-violet-600 dark:text-violet-400"
        >
          <span class="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-violet-500"></span>
          Live
        </span>
      </div>

      <EmptyState
        v-if="!tasks.length"
        title="No tasks yet"
        description="Queued work appears here and keeps running in the background, surviving restarts."
        icon="clock"
        color="violet"
      />

      <ul v-else class="space-y-3">
        <TaskCard
          v-for="task in tasks"
          :key="task.id"
          :task="task"
          @cancel="handleCancel"
          @delete="handleDelete"
        />
      </ul>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { AITaskStatus } from '@browser-server/shared-types'
import { useModal } from '@browser-server/shared-modal'
import { onMounted, ref } from 'vue'
import Button from './ui/Button.vue'
import EmptyState from './ui/EmptyState.vue'
import ErrorBanner from './ui/ErrorBanner.vue'
import LoadingSpinner from './ui/LoadingSpinner.vue'
import PageHeader from './ui/PageHeader.vue'
import StatCard from './ui/StatCard.vue'
import TaskCard from './tasks/TaskCard.vue'
import { useAITasks, type TaskFilter } from '../composables/useAITasks'

const {
  tasks,
  status,
  filter,
  isLoading,
  isSubmitting,
  error,
  enabled,
  workers,
  counts,
  hasActive,
  load,
  submit,
  cancel,
  remove,
  setFilter,
} = useAITasks()

const { confirm, confirmDelete } = useModal()
const draft = ref('')

const STATUSES: AITaskStatus[] = ['queued', 'running', 'completed', 'failed']

const filterOptions: { value: TaskFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  ...STATUSES.map((value) => ({ value, label: value[0].toUpperCase() + value.slice(1) })),
]

/** 'all' has no per-status count of its own — the header stats carry the totals. */
function countFor(value: TaskFilter): number | null {
  return value === 'all' ? null : (counts.value[value] ?? 0)
}

async function handleSubmit() {
  if (await submit(draft.value)) draft.value = ''
}

async function handleCancel(id: string) {
  if (await confirm('Cancel this task?', 'It will be marked failed and will not run.')) {
    await cancel(id)
  }
}

async function handleDelete(id: string) {
  if (await confirmDelete('Delete this task?', 'Its result and history will be removed.')) {
    await remove(id)
  }
}

onMounted(() => load())
</script>
