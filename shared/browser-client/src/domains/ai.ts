import type {
  AIConfig,
  AIConversation,
  AIConversationDetail,
  AIToolDecisionResponse,
  CreateAIConversationInput,
  SendAIMessageInput,
  SendAIMessageResponse,
  StopAIGenerationResponse,
  UpdateAIConversationInput,
} from '@browser-server/shared-types'
import { type TokenProvider, apiFetch, apiErrorFromBody, authHeader, buildQuery } from '../internals'

export function createAIMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    getAIConfig(): Promise<AIConfig> {
      return apiFetch<AIConfig>(baseUrl, 'GET', '/api/ai/config', undefined, getToken)
    },

    listAIConversations(query?: string, limit?: number): Promise<AIConversation[]> {
      return apiFetch<AIConversation[]>(
        baseUrl,
        'GET',
        `/api/ai/conversations${buildQuery({ q: query, limit })}`,
        undefined,
        getToken,
      )
    },

    createAIConversation(data: CreateAIConversationInput = {}): Promise<AIConversation> {
      return apiFetch<AIConversation>(baseUrl, 'POST', '/api/ai/conversations', data, getToken)
    },

    getAIConversation(id: string): Promise<AIConversationDetail> {
      return apiFetch<AIConversationDetail>(baseUrl, 'GET', `/api/ai/conversations/${encodeURIComponent(id)}`, undefined, getToken)
    },

    updateAIConversation(id: string, data: UpdateAIConversationInput): Promise<AIConversation> {
      return apiFetch<AIConversation>(baseUrl, 'PATCH', `/api/ai/conversations/${encodeURIComponent(id)}`, data, getToken)
    },

    deleteAIConversation(id: string): Promise<void> {
      return apiFetch<void>(baseUrl, 'DELETE', `/api/ai/conversations/${encodeURIComponent(id)}`, undefined, getToken)
    },

    sendAIMessage(id: string, data: SendAIMessageInput): Promise<SendAIMessageResponse> {
      return apiFetch<SendAIMessageResponse>(
        baseUrl,
        'POST',
        `/api/ai/conversations/${encodeURIComponent(id)}/messages`,
        { ...data, stream: false },
        getToken,
      )
    },

    /**
     * Send a message and consume the SSE stream. Returns an AbortController
     * that the caller can use to cancel.
     */
    sendAIMessageStream(
      id: string,
      data: SendAIMessageInput,
      onEvent: (event: import('@browser-server/shared-types').AIStreamEvent) => void,
      onError?: (err: Error) => void,
    ): AbortController {
      const controller = new AbortController()
      const url = `${baseUrl}/api/ai/conversations/${encodeURIComponent(id)}/messages`
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...authHeader(getToken),
      }

      fetch(url, {
        method: 'POST',
        headers,
        body: JSON.stringify({ ...data, stream: true }),
        signal: controller.signal,
      })
        .then(async (response) => {
          if (!response.ok) {
            const text = await response.text()
            throw apiErrorFromBody(response.status, text, `Stream failed: ${response.status}`)
          }
          const reader = response.body?.getReader()
          if (!reader) throw new Error('No response body')
          const decoder = new TextDecoder()
          let buffer = ''
          let streamEnded = false

          const processFrames = () => {
            let boundary = buffer.indexOf('\n\n')
            while (boundary >= 0) {
              const frame = buffer.slice(0, boundary)
              buffer = buffer.slice(boundary + 2)
              let eventType = ''
              const dataLines: string[] = []
              for (const line of frame.split('\n')) {
                if (line.startsWith('event:')) eventType = line.slice(6).trim()
                else if (line.startsWith('data:')) dataLines.push(line.slice(5).trimStart())
              }
              if (eventType && dataLines.length > 0) {
                const parsed = JSON.parse(dataLines.join('\n'))
                onEvent({ type: eventType, ...parsed } as import('@browser-server/shared-types').AIStreamEvent)
                if (eventType === 'done' || eventType === 'error') streamEnded = true
              }
              boundary = buffer.indexOf('\n\n')
            }
          }

          while (true) {
            const { done, value } = await reader.read()
            if (done) break
            buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, '\n')
            processFrames()
            if (streamEnded) break
          }
          buffer += decoder.decode().replace(/\r\n/g, '\n')
          processFrames()
          if (!streamEnded) throw new Error('AI stream ended before a terminal event')
          reader.cancel().catch(() => {})
        })
        .catch((err) => {
          if (err.name === 'AbortError') return
          onError?.(err instanceof Error ? err : new Error(String(err)))
        })

      return controller
    },

    regenerateAIMessage(id: string): Promise<SendAIMessageResponse> {
      return apiFetch<SendAIMessageResponse>(
        baseUrl,
        'POST',
        `/api/ai/conversations/${encodeURIComponent(id)}/regenerate`,
        {},
        getToken,
      )
    },

    decideAIToolCall(id: string, callId: string, approved: boolean): Promise<AIToolDecisionResponse> {
      return apiFetch<AIToolDecisionResponse>(
        baseUrl,
        'POST',
        `/api/ai/conversations/${encodeURIComponent(id)}/tool-calls/${encodeURIComponent(callId)}`,
        { approved },
        getToken,
      )
    },

    stopAIGeneration(id: string): Promise<StopAIGenerationResponse> {
      return apiFetch<StopAIGenerationResponse>(
        baseUrl,
        'POST',
        `/api/ai/conversations/${encodeURIComponent(id)}/stop`,
        {},
        getToken,
      )
    },
  }
}
