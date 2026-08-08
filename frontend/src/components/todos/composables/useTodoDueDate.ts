import type { DueDateFilter } from '../../../types';
import { ref, type Ref } from 'vue';

/** Due-date filter state. Predicates and labels live in `../todoFormat`. */
export function useTodoDueDate() {
  const dueDateFilter: Ref<DueDateFilter> = ref(null);

  function clearDueDateFilter() {
    dueDateFilter.value = null;
  }

  return { dueDateFilter, clearDueDateFilter };
}
