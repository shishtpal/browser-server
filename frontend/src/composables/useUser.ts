import { ref, watch } from 'vue'
import { getUsers, getUser } from '../lib/api'
import type { User } from '../types'

const USER_STORAGE_KEY = 'browser-server-user-id'

const currentUserId = ref<number | null>(loadUserId())
const currentUser = ref<User | null>(null)
const users = ref<User[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

function loadUserId(): number | null {
  try {
    const stored = localStorage.getItem(USER_STORAGE_KEY)
    return stored ? Number(stored) : null
  } catch {
    return null
  }
}

export function useUser() {
  const fetchUsers = async () => {
    isLoading.value = true
    error.value = null
    try {
      users.value = await getUsers()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load users'
    } finally {
      isLoading.value = false
    }
  }

  const fetchCurrentUser = async () => {
    if (!currentUserId.value) {
      currentUser.value = null
      return
    }
    try {
      currentUser.value = await getUser(currentUserId.value)
    } catch {
      currentUser.value = null
    }
  }

  const setUser = (id: number) => {
    currentUserId.value = id
    localStorage.setItem(USER_STORAGE_KEY, String(id))
    fetchCurrentUser()
  }

  const clearUser = () => {
    currentUserId.value = null
    currentUser.value = null
    localStorage.removeItem(USER_STORAGE_KEY)
  }

  watch(currentUserId, () => {
    fetchCurrentUser()
  })

  // Initial load
  fetchUsers()
  if (currentUserId.value) {
    fetchCurrentUser()
  }

  return {
    currentUserId,
    currentUser,
    users,
    isLoading,
    error,
    setUser,
    clearUser,
    fetchUsers,
  }
}
