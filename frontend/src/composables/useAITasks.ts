import type { AITask, AITaskStatus, AITaskStatusResponse } from '@browser-server/shared-types'
import { computed, onUnmounted, ref } from 'vue'
import { cancelAITask, createAITask, deleteAITask, getAITaskStatus, listAITasks } from '../lib/api'

export type TaskFilter = AITaskStatus | 'all'

/**
 * Polling interval while any task is queued or running. The server has no
 * /ws/tasks stream, so progress is only visible by re-reading the list; three
 * seconds is frequent enough to feel live without hammering a queue that is
 * usually idle.
 */
const ACTIVE_POLL_MS = 3000

export function useAITasks() {
  const tasks = ref<AITask[]>([])
  const status = ref<AITaskStatusResponse | null>(null)
  const filter = ref<TaskFilter>('all')
  const isLoading = ref(false)
  const isSubmitting = ref(false)
  const error = ref('')

  let timer: ReturnType<typeof setTimeout> | null = null

  const enabled = computed(() => status.value?.enabled ?? false)
  const workers = computed(() => status.value?.workers ?? 0)
  const counts = computed(() => status.value?.counts ?? ({} as Record<AITaskStatus, number>))

  /**
   * Whether anything is still moving — the only reason to keep polling. Driven
   * by the status counts rather than the visible list, so a task running while
   * the "Completed" filter is selected still refreshes the view it will land in.
   */
  const hasActive = computed(
    () => (counts.value.queued ?? 0) > 0 || (counts.value.running ?? 0) > 0,
  )

  function stopPolling() {
    if (timer !== null) {
      clearTimeout(timer)
      timer = null
    }
  }

  /**
   * Schedules the next poll only while work is in flight. Chained timeouts
   * rather than an interval so a slow response cannot stack requests.
   */
  function schedulePoll() {
    stopPolling()
    if (!enabled.value || !hasActive.value) return
    timer = setTimeout(() => {
      void load(true)
    }, ACTIVE_POLL_MS)
  }

  async function load(silent = false) {
    if (!silent) isLoading.value = true
    try {
      status.value = await getAITaskStatus()
      if (status.value.enabled) {
        tasks.value = await listAITasks(filter.value === 'all' ? undefined : filter.value)
      } else {
        tasks.value = []
      }
      error.value = ''
    } catch (err) {
      // A background refresh that fails should not blank out a list the user is
      // reading; only a foreground load reports the failure.
      if (!silent) error.value = err instanceof Error ? err.message : 'Failed to load tasks'
    } finally {
      isLoading.value = false
      schedulePoll()
    }
  }

  async function submit(prompt: string, conversationId?: string): Promise<boolean> {
    const trimmed = prompt.trim()
    if (!trimmed || isSubmitting.value) return false
    isSubmitting.value = true
    try {
      await createAITask({ prompt: trimmed, conversation_id: conversationId || undefined })
      await load(true)
      error.value = ''
      return true
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create task'
      return false
    } finally {
      isSubmitting.value = false
    }
  }

  async function cancel(id: string) {
    try {
      await cancelAITask(id)
      await load(true)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to cancel task'
    }
  }

  async function remove(id: string) {
    try {
      await deleteAITask(id)
      await load(true)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete task'
    }
  }

  function setFilter(next: TaskFilter) {
    filter.value = next
    void load()
  }

  onUnmounted(stopPolling)

  return {
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
  }
}
