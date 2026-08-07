import { ref, watch, type Ref } from 'vue'
import { generatePaper, getPapers, getPaper, deletePaper } from '../lib/api'
import type { GeneratePaperInput, QuestionPaper } from '../types'

export function useQuizPapers(userId: Ref<number | null>) {
  const papers = ref<QuestionPaper[]>([])
  const isLoading = ref(false)
  const isGenerating = ref(false)
  const error = ref<string | null>(null)
  const activePaper = ref<QuestionPaper | null>(null)

  const loadPapers = async () => {
    if (!userId.value) return
    isLoading.value = true
    error.value = null
    try {
      papers.value = await getPapers(userId.value)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load papers'
    } finally {
      isLoading.value = false
    }
  }

  const generate = async (input: Omit<GeneratePaperInput, 'user_id'>) => {
    if (!userId.value) return undefined
    isGenerating.value = true
    error.value = null
    try {
      const paper = await generatePaper({ ...input, user_id: userId.value })
      papers.value.unshift({ ...paper, questions: undefined })
      return paper
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to generate paper'
      return undefined
    } finally {
      isGenerating.value = false
    }
  }

  const openPaper = async (id: number) => {
    error.value = null
    try {
      activePaper.value = await getPaper(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load paper'
    }
  }

  const closePaper = () => {
    activePaper.value = null
  }

  const removePaper = async (id: number) => {
    try {
      await deletePaper(id)
      papers.value = papers.value.filter((p) => p.id !== id)
      if (activePaper.value?.id === id) activePaper.value = null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete paper'
    }
  }

  watch(userId, (val) => {
    if (val && val > 0) {
      loadPapers()
    } else {
      papers.value = []
      activePaper.value = null
    }
  })

  return {
    papers,
    isLoading,
    isGenerating,
    error,
    activePaper,
    loadPapers,
    generate,
    openPaper,
    closePaper,
    removePaper,
  }
}
