<template>
  <div class="mx-auto max-w-full px-3 py-4 sm:px-6 lg:px-10 xl:px-12">
    <PageHeader badge="Background agent" title="Tasks" color="violet">
      <template #stats>
        <StatCard :value="counts.running ?? 0" label="Running" variant="primary" color="violet" />
        <StatCard :value="counts.queued ?? 0" label="Queued" variant="dark" color="violet" />
      </template>
      <template #controls>
        <Button variant="secondary" size="sm" :disabled="isLoading" @click="load()">
          <span class="inline-flex items-center gap-1.5">
            <RefreshCw class="h-3.5 w-3.5" :stroke-width="2.5" aria-hidden="true" />
            Refresh
          </span>
        </Button>
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
      <TaskSubmitForm :workers="workers" :is-submitting="isSubmitting" @submit="submitTask" />

      <TaskFilterBar :filter="filter" :counts="counts" :live="hasActive" @change="setFilter" />

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
          @cancel="cancelTask"
          @delete="deleteTask"
        />
      </ul>
    </template>
  </div>
</template>

<script setup lang="ts">
import { RefreshCw } from '@lucide/vue';
import { useTasksPage } from './tasks/composables/useTasksPage';
import Button from './ui/Button.vue';
import EmptyState from './ui/EmptyState.vue';
import ErrorBanner from './ui/ErrorBanner.vue';
import LoadingSpinner from './ui/LoadingSpinner.vue';
import PageHeader from './ui/PageHeader.vue';
import StatCard from './ui/StatCard.vue';
import TaskCard from './tasks/TaskCard.vue';
import TaskFilterBar from './tasks/TaskFilterBar.vue';
import TaskSubmitForm from './tasks/TaskSubmitForm.vue';

const { tasksApi, submitTask, cancelTask, deleteTask } = useTasksPage();

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
  setFilter,
} = tasksApi;
</script>
