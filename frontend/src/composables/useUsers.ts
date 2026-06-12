import { ref } from 'vue'
import { getUsers, createUser, deleteUser } from '../lib/api'
import type { User } from '../types'

export function useUsers() {
  const usersList = ref<User[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const successMsg = ref('')

  const newUsername = ref('')
  const newEmail = ref('')

  const loadUsers = async () => {
    isLoading.value = true
    error.value = null
    try {
      usersList.value = await getUsers()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load users'
    } finally {
      isLoading.value = false
    }
  }

  const addUser = async () => {
    if (!newUsername.value.trim()) return
    const username = newUsername.value.trim()
    try {
      await createUser({
        username,
        email: newEmail.value.trim() || undefined,
      })
      successMsg.value = `User "${username}" created!`
      newUsername.value = ''
      newEmail.value = ''
      await loadUsers()
      setTimeout(() => { successMsg.value = '' }, 3000)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create user'
    }
  }

  const removeUser = async (id: number) => {
    if (!confirm('Delete this user? This will remove all their data (todos, bookmarks, history, wallet entries).')) return
    try {
      await deleteUser(id)
      await loadUsers()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete user'
    }
  }

  return {
    usersList,
    isLoading,
    error,
    successMsg,
    newUsername,
    newEmail,
    loadUsers,
    addUser,
    removeUser,
  }
}
