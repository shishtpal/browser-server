import { onMounted } from 'vue';
import { useModal } from '@browser-server/shared-modal';
import { useAITasks } from './useAITasks';

/**
 * Orchestrates the Tasks page: queued-task polling, submit/cancel/delete with
 * shared-modal confirmations, and initial load — so TasksPage.vue stays wiring.
 */
export function useTasksPage() {
  const tasksApi = useAITasks();
  const { confirm, confirmDelete } = useModal();

  async function submitTask(prompt: string) {
    await tasksApi.submit(prompt);
  }

  async function cancelTask(id: string) {
    if (await confirm('Cancel this task?', 'It will be marked failed and will not run.')) {
      await tasksApi.cancel(id);
    }
  }

  async function deleteTask(id: string) {
    if (await confirmDelete('Delete this task?', 'Its result and history will be removed.')) {
      await tasksApi.remove(id);
    }
  }

  onMounted(() => tasksApi.load());

  return {
    tasksApi,
    submitTask,
    cancelTask,
    deleteTask,
  };
}
