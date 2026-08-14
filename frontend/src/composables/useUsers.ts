import { onScopeDispose, ref } from 'vue';
import { getUsers, createUser, deleteUser } from '../lib/api';
import type { User } from '../types';

export function useUsers() {
  const usersList = ref<User[]>([]);
  const isLoading = ref(false);
  const isCreating = ref(false);
  const deletingId = ref<number | null>(null);
  const error = ref<string | null>(null);
  const successMsg = ref('');

  const newUsername = ref('');
  const newEmail = ref('');
  let successTimer: number | undefined;

  function showSuccess(message: string) {
    successMsg.value = message;
    if (successTimer) window.clearTimeout(successTimer);
    successTimer = window.setTimeout(() => (successMsg.value = ''), 3000);
  }

  const loadUsers = async () => {
    isLoading.value = true;
    error.value = null;
    try {
      usersList.value = await getUsers();
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Failed to load users';
    } finally {
      isLoading.value = false;
    }
  };

  const addUser = async (): Promise<User | null> => {
    if (!newUsername.value.trim() || isCreating.value) return null;
    const username = newUsername.value.trim();
    isCreating.value = true;
    error.value = null;
    try {
      const created = await createUser({
        username,
        email: newEmail.value.trim() || undefined,
      });
      newUsername.value = '';
      newEmail.value = '';
      await loadUsers();
      showSuccess(`Workspace “${username}” created.`);
      return created;
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Failed to create user';
      return null;
    } finally {
      isCreating.value = false;
    }
  };

  const removeUser = async (id: number): Promise<boolean> => {
    if (deletingId.value !== null) return false;
    deletingId.value = id;
    error.value = null;
    try {
      await deleteUser(id);
      await loadUsers();
      showSuccess('Workspace deleted.');
      return true;
    } catch (caught) {
      error.value = caught instanceof Error ? caught.message : 'Failed to delete user';
      return false;
    } finally {
      deletingId.value = null;
    }
  };

  onScopeDispose(() => {
    if (successTimer) window.clearTimeout(successTimer);
  });

  return {
    usersList,
    isLoading,
    isCreating,
    deletingId,
    error,
    successMsg,
    newUsername,
    newEmail,
    loadUsers,
    addUser,
    removeUser,
  };
}
