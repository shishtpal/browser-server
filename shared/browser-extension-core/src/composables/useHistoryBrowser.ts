import type { GroupedHistoryEntry, HistoryDomainSummary } from '@browser-server/shared-client'
import { computed, nextTick, ref, watch, type ComputedRef, type Ref } from 'vue'
import { createApiClient, useUserId } from './useApiClient'
import { useExtensionSettings } from './useExtensionSettings'

const PAGE_SIZE = 100

export interface UseHistoryBrowserReturn {
  settings: ReturnType<typeof useExtensionSettings>['settings']
  userId: ComputedRef<number>
  domains: Ref<HistoryDomainSummary[]>
  entries: Ref<GroupedHistoryEntry[]>
  selectedDomain: Ref<string>
  selectedEntry: Ref<GroupedHistoryEntry | null>
  domainFilter: Ref<string>
  linkFilter: Ref<string>
  domainLoading: Ref<boolean>
  linksLoading: Ref<boolean>
  errorMessage: Ref<string>
  totalEntries: Ref<number>
  currentPage: Ref<number>
  filteredDomains: ComputedRef<HistoryDomainSummary[]>
  totalPages: ComputedRef<number>
  isReady: ComputedRef<boolean>
  loadDomains(): Promise<void>
  loadEntries(): Promise<void>
  refresh(): Promise<void>
  selectDomain(domain: string): void
  selectEntry(entry: GroupedHistoryEntry): void
  changePage(delta: number): void
  cleanup(): void
}

export function useHistoryBrowser(
  onEntrySelected?: (entry: GroupedHistoryEntry) => void,
  onLayoutReady?: () => void,
): UseHistoryBrowserReturn {
  const { settings } = useExtensionSettings()
  const userId = useUserId(computed(() => settings.value))
  const client = computed(() => (settings.value ? createApiClient(settings.value) : null))

  const domains = ref<HistoryDomainSummary[]>([])
  const entries = ref<GroupedHistoryEntry[]>([])
  const selectedDomain = ref('')
  const selectedEntry = ref<GroupedHistoryEntry | null>(null)
  const domainFilter = ref('')
  const linkFilter = ref('')
  const debouncedLinkFilter = ref('')
  const domainLoading = ref(false)
  const linksLoading = ref(false)
  const errorMessage = ref('')
  const totalEntries = ref(0)
  const currentPage = ref(1)

  let linkFilterTimer: ReturnType<typeof setTimeout> | undefined
  let domainRequest = 0
  let linksRequest = 0

  const filteredDomains = computed(() => {
    const query = domainFilter.value.trim().toLowerCase()
    return query ? domains.value.filter((item) => item.domain.includes(query)) : domains.value
  })
  const totalPages = computed(() => Math.max(1, Math.ceil(totalEntries.value / PAGE_SIZE)))
  const isReady = computed(() => Boolean(client.value) && userId.value > 0)

  async function loadDomains() {
    if (!client.value || !userId.value) return
    const request = ++domainRequest
    domainLoading.value = true
    errorMessage.value = ''
    try {
      const result = await client.value.getHistoryDomains(userId.value)
      if (request !== domainRequest) return
      domains.value = result
      if (!selectedDomain.value || !result.some((item) => item.domain === selectedDomain.value)) {
        selectDomain(result[0]?.domain ?? '')
      }
    } catch (error) {
      if (request !== domainRequest) return
      domains.value = []
      errorMessage.value = error instanceof Error ? error.message : 'Could not load history.'
    } finally {
      if (request === domainRequest) domainLoading.value = false
    }
  }

  async function loadEntries() {
    const request = ++linksRequest
    if (!client.value || !userId.value || !selectedDomain.value) {
      entries.value = []
      totalEntries.value = 0
      return
    }
    linksLoading.value = true
    errorMessage.value = ''
    try {
      const result = await client.value.getGroupedHistory({
        user_id: userId.value,
        domain: selectedDomain.value,
        q: debouncedLinkFilter.value.trim() || undefined,
        column: 'all',
        limit: PAGE_SIZE,
        offset: (currentPage.value - 1) * PAGE_SIZE,
      })
      if (request !== linksRequest) return
      entries.value = result.entries
      totalEntries.value = result.total
    } catch (error) {
      if (request !== linksRequest) return
      entries.value = []
      totalEntries.value = 0
      errorMessage.value = error instanceof Error ? error.message : 'Could not load links.'
    } finally {
      if (request === linksRequest) linksLoading.value = false
    }
  }

  async function refresh() {
    await loadDomains()
    await loadEntries()
  }

  function selectDomain(domain: string) {
    if (selectedDomain.value === domain) return
    selectedDomain.value = domain
    selectedEntry.value = null
    linkFilter.value = ''
    debouncedLinkFilter.value = ''
    currentPage.value = 1
    void loadEntries()
  }

  function selectEntry(entry: GroupedHistoryEntry) {
    selectedEntry.value = entry
    onEntrySelected?.(entry)
  }

  function changePage(delta: number) {
    const next = currentPage.value + delta
    if (next < 1 || next > totalPages.value) return
    currentPage.value = next
    void loadEntries()
  }

  function cleanup() {
    if (linkFilterTimer) clearTimeout(linkFilterTimer)
  }

  watch(linkFilter, (value) => {
    if (linkFilterTimer) clearTimeout(linkFilterTimer)
    linkFilterTimer = setTimeout(() => {
      debouncedLinkFilter.value = value
      currentPage.value = 1
      void loadEntries()
    }, 200)
  })

  watch([client, userId], () => {
    domainRequest++
    linksRequest++
    domains.value = []
    entries.value = []
    selectedDomain.value = ''
    selectedEntry.value = null
    totalEntries.value = 0
    domainLoading.value = false
    linksLoading.value = false
    if (isReady.value) {
      void loadDomains()
      if (onLayoutReady) void nextTick(onLayoutReady)
    }
  }, { immediate: true })

  return {
    settings,
    userId,
    domains,
    entries,
    selectedDomain,
    selectedEntry,
    domainFilter,
    linkFilter,
    domainLoading,
    linksLoading,
    errorMessage,
    totalEntries,
    currentPage,
    filteredDomains,
    totalPages,
    isReady,
    loadDomains,
    loadEntries,
    refresh,
    selectDomain,
    selectEntry,
    changePage,
    cleanup,
  }
}
