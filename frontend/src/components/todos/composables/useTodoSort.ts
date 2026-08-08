import type { Todo, TodoSortField } from '../../../types';
import { computed, ref, type ComputedRef, type Ref } from 'vue';
import { priorityWeight } from '../todoFormat';

export function useTodoSort(sourceTodos: Ref<Todo[]>) {
  const sortField: Ref<TodoSortField> = ref('position');
  const sortDir: Ref<'asc' | 'desc'> = ref('asc');

  const sorted: ComputedRef<Todo[]> = computed(() => {
    const list = [...sourceTodos.value];
    const field = sortField.value;
    const dir = sortDir.value === 'asc' ? 1 : -1;

    list.sort((a, b) => {
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;

      let cmp = 0;
      switch (field) {
        case 'position':
          cmp = a.position - b.position;
          break;
        case 'priority':
          cmp = priorityWeight(a.priority) - priorityWeight(b.priority);
          break;
        case 'start_date': {
          const ad = a.start_date ? new Date(a.start_date).getTime() : Infinity;
          const bd = b.start_date ? new Date(b.start_date).getTime() : Infinity;
          cmp = ad - bd;
          break;
        }
        case 'created_at': {
          const ac = new Date(a.created_at).getTime();
          const bc = new Date(b.created_at).getTime();
          cmp = ac - bc;
          break;
        }
        case 'title':
          cmp = a.title.localeCompare(b.title);
          break;
      }
      return cmp * dir;
    });
    return list;
  });

  function toggleDir() {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc';
  }

  function setSort(field: TodoSortField) {
    if (sortField.value === field) {
      toggleDir();
    } else {
      sortField.value = field;
      sortDir.value = 'asc';
    }
  }

  return {
    sortField,
    sortDir,
    sorted,
    setSort,
    toggleDir,
  };
}
