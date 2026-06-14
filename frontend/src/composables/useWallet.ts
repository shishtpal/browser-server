import { ref, computed, watch, type Ref } from 'vue'
import { getWallet, createWalletEntry, updateWalletEntry, deleteWalletEntry, revealWalletPassword } from '../lib/api'
import type { WalletEntry } from '../types'

export function useWallet(selectedUserId: Ref<number | null>) {
  const walletEntries = ref<WalletEntry[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const websiteFilter = ref('')
  const searchColumn = ref<'website' | 'username' | 'description' | 'all'>('website')

  const newWebsite = ref('')
  const newUsername = ref('')
  const newPassword = ref('')
  const newDescription = ref('')

  const editing = ref<WalletEntry | null>(null)
  const editForm = ref({ website: '', username: '', password: '', description: '' })

  const filteredEntries = computed(() => {
    if (!websiteFilter.value.trim()) return walletEntries.value
    const q = websiteFilter.value.toLowerCase()
    const col = searchColumn.value
    return walletEntries.value.filter(e => {
      if (col === 'all') {
        return e.website.toLowerCase().includes(q) ||
               e.username.toLowerCase().includes(q) ||
               e.description.toLowerCase().includes(q)
      }
      if (col === 'website') return e.website.toLowerCase().includes(q)
      if (col === 'username') return e.username.toLowerCase().includes(q)
      if (col === 'description') return e.description.toLowerCase().includes(q)
      return false
    })
  })

  const loadWallet = async () => {
    if (!selectedUserId.value) return
    isLoading.value = true
    error.value = null
    try {
      walletEntries.value = await getWallet(selectedUserId.value)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load wallet'
    } finally {
      isLoading.value = false
    }
  }

  const addEntry = async () => {
    if (!selectedUserId.value || !newWebsite.value.trim() || !newUsername.value.trim() || !newPassword.value) return
    try {
      await createWalletEntry({
        user_id: selectedUserId.value,
        website: newWebsite.value.trim(),
        username: newUsername.value.trim(),
        password: newPassword.value,
        description: newDescription.value.trim() || undefined,
      })
      newWebsite.value = ''
      newUsername.value = ''
      newPassword.value = ''
      newDescription.value = ''
      await loadWallet()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add entry'
    }
  }

  const openEdit = async (e: WalletEntry) => {
    editing.value = e
    editForm.value = {
      website: e.website,
      username: e.username,
      password: '',
      description: e.description,
    }
    // Passwords are not included in the list response; fetch on demand.
    if (selectedUserId.value) {
      try {
        editForm.value.password = await revealWalletPassword(selectedUserId.value, e.website, e.username)
      } catch {
        editForm.value.password = ''
      }
    }
  }

  const saveEdit = async () => {
    if (!editing.value) return
    try {
      await updateWalletEntry(editing.value.id, { ...editing.value, ...editForm.value })
      editing.value = null
      await loadWallet()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update entry'
    }
  }

  const removeEntry = async (id: number) => {
    if (!confirm('Delete this wallet entry?')) return
    try {
      await deleteWalletEntry(id)
      await loadWallet()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete entry'
    }
  }

  return {
    walletEntries,
    isLoading,
    error,
    websiteFilter,
    searchColumn,
    newWebsite,
    newUsername,
    newPassword,
    newDescription,
    editing,
    editForm,
    filteredEntries,
    loadWallet,
    addEntry,
    openEdit,
    saveEdit,
    removeEntry,
  }
}
