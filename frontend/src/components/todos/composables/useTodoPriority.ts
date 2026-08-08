import { ref, type Ref } from 'vue';
import type { TodoPriority } from '../../../types';

/** Priority filter state. Presentation metadata lives in `../todoFormat`. */
export function useTodoPriority() {
  const selectedPriority: Ref<TodoPriority | null> = ref(null);

  function clearPriority() {
    selectedPriority.value = null;
  }

  return { selectedPriority, clearPriority };
}
