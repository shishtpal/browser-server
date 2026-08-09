import type {
  AIConfig,
  AIVoiceConfig,
  AIConversation,
  AIConversationDetail,
  AIImageAttachment,
  AIAttachmentSummary,
  AIMessage,
  AIMonitoring,
  AIRequestLogList,
  AITask,
  AITaskStatus,
  AITaskStatusResponse,
  AIToolDecisionResponse,
  AppendAIMessageInput,
  CreateAIConversationInput,
  CreateAITaskInput,
  CreateAITaskResponse,
  ForkAIConversationInput,
  SendAIMessageInput,
  SendAIMessageResponse,
  StopAIGenerationResponse,
  UpdateAIConversationInput,
  AIImageConfig,
  GeneratedImage,
  GenerateImageInput,
  GenerateImageResponse,
  AIMemoryStats,
  AIMemoryGraph,
  AIMemoryFragment,
  AIMemoryWriteResult,
  AIMemoryWriteOp,
} from "@browser-server/shared-types";
import {
  type TokenProvider,
  apiFetch,
  apiErrorFromBody,
  authHeader,
  buildQuery,
} from "../internals";

export function createAIMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    getAIConfig(): Promise<AIConfig> {
      return apiFetch<AIConfig>(baseUrl, "GET", "/api/ai/config", undefined, getToken);
    },

    getAIImageConfig(): Promise<AIImageConfig> {
      return apiFetch<AIImageConfig>(baseUrl, "GET", "/api/ai/images/config", undefined, getToken);
    },
    listGeneratedImages(limit?: number): Promise<GeneratedImage[]> {
      return apiFetch<GeneratedImage[]>(
        baseUrl,
        "GET",
        `/api/ai/images${buildQuery({ limit })}`,
        undefined,
        getToken,
      );
    },
    generateImage(data: GenerateImageInput): Promise<GenerateImageResponse> {
      return apiFetch<GenerateImageResponse>(baseUrl, "POST", "/api/ai/images", data, getToken);
    },
    deleteGeneratedImage(id: string): Promise<void> {
      return apiFetch<void>(
        baseUrl,
        "DELETE",
        `/api/ai/images/${encodeURIComponent(id)}`,
        undefined,
        getToken,
      );
    },
    getGeneratedImageUrl(id: string): string {
      const token = getToken?.();
      const q = token ? `?token=${encodeURIComponent(token)}` : "";
      return `${baseUrl}/api/ai/images/${encodeURIComponent(id)}/file${q}`;
    },

    getAIVoiceConfig(): Promise<AIVoiceConfig> {
      return apiFetch<AIVoiceConfig>(baseUrl, "GET", "/api/ai/voice/config", undefined, getToken);
    },

    getAIRequestLogs(
      filters: {
        source?: "chat" | "task_agent";
        status?: "success" | "error" | "cancelled";
        conversationId?: string;
        taskId?: string;
        limit?: number;
        offset?: number;
      } = {},
    ): Promise<AIRequestLogList> {
      return apiFetch<AIRequestLogList>(
        baseUrl,
        "GET",
        `/api/ai/logs${buildQuery({
          source: filters.source,
          status: filters.status,
          conversation_id: filters.conversationId,
          task_id: filters.taskId,
          limit: filters.limit,
          offset: filters.offset,
        })}`,
        undefined,
        getToken,
      );
    },

    getAIMonitoring(windowHours?: number): Promise<AIMonitoring> {
      return apiFetch<AIMonitoring>(
        baseUrl,
        "GET",
        `/api/ai/monitoring${buildQuery({ window_hours: windowHours })}`,
        undefined,
        getToken,
      );
    },

    listAIConversations(query?: string, limit?: number): Promise<AIConversation[]> {
      return apiFetch<AIConversation[]>(
        baseUrl,
        "GET",
        `/api/ai/conversations${buildQuery({ q: query, limit })}`,
        undefined,
        getToken,
      );
    },

    createAIConversation(data: CreateAIConversationInput = {}): Promise<AIConversation> {
      return apiFetch<AIConversation>(baseUrl, "POST", "/api/ai/conversations", data, getToken);
    },

    forkAIConversation(id: string, data: ForkAIConversationInput): Promise<AIConversation> {
      return apiFetch<AIConversation>(
        baseUrl,
        "POST",
        `/api/ai/conversations/${encodeURIComponent(id)}/fork`,
        data,
        getToken,
      );
    },

    getAIConversation(id: string): Promise<AIConversationDetail> {
      return apiFetch<AIConversationDetail>(
        baseUrl,
        "GET",
        `/api/ai/conversations/${encodeURIComponent(id)}`,
        undefined,
        getToken,
      );
    },

    updateAIConversation(id: string, data: UpdateAIConversationInput): Promise<AIConversation> {
      return apiFetch<AIConversation>(
        baseUrl,
        "PATCH",
        `/api/ai/conversations/${encodeURIComponent(id)}`,
        data,
        getToken,
      );
    },

    deleteAIConversation(id: string): Promise<void> {
      return apiFetch<void>(
        baseUrl,
        "DELETE",
        `/api/ai/conversations/${encodeURIComponent(id)}`,
        undefined,
        getToken,
      );
    },

    updateAIMessage(
      conversationId: string,
      messageId: string,
      data: import("@browser-server/shared-types").UpdateAIMessageInput,
    ): Promise<import("@browser-server/shared-types").AIMessage> {
      return apiFetch<import("@browser-server/shared-types").AIMessage>(
        baseUrl,
        "PATCH",
        `/api/ai/conversations/${encodeURIComponent(conversationId)}/messages/${encodeURIComponent(messageId)}`,
        data,
        getToken,
      );
    },

    deleteAIMessage(conversationId: string, messageId: string): Promise<void> {
      return apiFetch<void>(
        baseUrl,
        "DELETE",
        `/api/ai/conversations/${encodeURIComponent(conversationId)}/messages/${encodeURIComponent(messageId)}`,
        undefined,
        getToken,
      );
    },

    sendAIMessage(id: string, data: SendAIMessageInput): Promise<SendAIMessageResponse> {
      return apiFetch<SendAIMessageResponse>(
        baseUrl,
        "POST",
        `/api/ai/conversations/${encodeURIComponent(id)}/messages`,
        { ...data, stream: false },
        getToken,
      );
    },

    async uploadAIImageAttachment(
      id: string,
      file: Blob,
      filename?: string,
    ): Promise<AIImageAttachment> {
      const formData = new FormData();
      formData.append("file", file, filename || "image");
      const response = await fetch(
        `${baseUrl}/api/ai/conversations/${encodeURIComponent(id)}/attachments`,
        {
          method: "POST",
          headers: authHeader(getToken),
          body: formData,
        },
      );
      if (!response.ok) {
        const text = await response.text();
        throw apiErrorFromBody(response.status, text, `Upload failed: ${response.status}`);
      }
      return response.json() as Promise<AIImageAttachment>;
    },

    deleteAIImageAttachment(id: string, attachmentId: string): Promise<void> {
      return apiFetch<void>(
        baseUrl,
        "DELETE",
        `/api/ai/conversations/${encodeURIComponent(id)}/attachments/${encodeURIComponent(attachmentId)}`,
        undefined,
        getToken,
      );
    },

    renameAIImageAttachment(
      id: string,
      attachmentId: string,
      filename: string,
    ): Promise<AIAttachmentSummary> {
      return apiFetch<AIAttachmentSummary>(
        baseUrl,
        "PATCH",
        `/api/ai/conversations/${encodeURIComponent(id)}/attachments/${encodeURIComponent(attachmentId)}`,
        { filename },
        getToken,
      );
    },

    getAIImageAttachmentUrl(id: string, attachmentId: string): string {
      // Image loads via <img src>, which can't set an Authorization header,
      // so the token is passed as a query param instead.
      const token = getToken?.();
      const suffix = token ? `?token=${encodeURIComponent(token)}` : "";
      return `${baseUrl}/api/ai/conversations/${encodeURIComponent(id)}/attachments/${encodeURIComponent(attachmentId)}${suffix}`;
    },

    /**
     * Fetch the raw attachment bytes as a Blob for programmatic reuse (e.g.
     * re-staging into another conversation). Unlike the <img> URL above, this
     * sends the token via the Authorization header, so it never appears in the
     * URL, and non-OK responses surface the server's error envelope as ApiError.
     */
    async getAIImageAttachmentBlob(id: string, attachmentId: string): Promise<Blob> {
      const response = await fetch(
        `${baseUrl}/api/ai/conversations/${encodeURIComponent(id)}/attachments/${encodeURIComponent(attachmentId)}`,
        { headers: authHeader(getToken) },
      );
      if (!response.ok) {
        const text = await response.text();
        throw apiErrorFromBody(
          response.status,
          text,
          `Failed to fetch attachment: ${response.status}`,
        );
      }
      return response.blob();
    },

    listAIAttachments(limit?: number): Promise<AIAttachmentSummary[]> {
      return apiFetch<AIAttachmentSummary[]>(
        baseUrl,
        "GET",
        `/api/ai/attachments${buildQuery({ limit })}`,
        undefined,
        getToken,
      );
    },

    appendAIMessage(id: string, data: AppendAIMessageInput): Promise<AIMessage> {
      return apiFetch<AIMessage>(
        baseUrl,
        "POST",
        `/api/ai/conversations/${encodeURIComponent(id)}/messages/append`,
        data,
        getToken,
      );
    },

    /**
     * Send a message and consume the SSE stream. Returns an AbortController
     * that the caller can use to cancel.
     */
    sendAIMessageStream(
      id: string,
      data: SendAIMessageInput,
      onEvent: (event: import("@browser-server/shared-types").AIStreamEvent) => void,
      onError?: (err: Error) => void,
    ): AbortController {
      const controller = new AbortController();
      const url = `${baseUrl}/api/ai/conversations/${encodeURIComponent(id)}/messages`;
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
        ...authHeader(getToken),
      };

      fetch(url, {
        method: "POST",
        headers,
        body: JSON.stringify({ ...data, stream: true }),
        signal: controller.signal,
      })
        .then(async (response) => {
          if (!response.ok) {
            const text = await response.text();
            throw apiErrorFromBody(response.status, text, `Stream failed: ${response.status}`);
          }
          const reader = response.body?.getReader();
          if (!reader) throw new Error("No response body");
          const decoder = new TextDecoder();
          let buffer = "";
          let streamEnded = false;

          const processFrames = () => {
            let boundary = buffer.indexOf("\n\n");
            while (boundary >= 0) {
              const frame = buffer.slice(0, boundary);
              buffer = buffer.slice(boundary + 2);
              let eventType = "";
              const dataLines: string[] = [];
              for (const line of frame.split("\n")) {
                if (line.startsWith("event:")) eventType = line.slice(6).trim();
                else if (line.startsWith("data:")) dataLines.push(line.slice(5).trimStart());
              }
              if (eventType && dataLines.length > 0) {
                const parsed = JSON.parse(dataLines.join("\n"));
                onEvent({
                  type: eventType,
                  ...parsed,
                } as import("@browser-server/shared-types").AIStreamEvent);
                if (eventType === "done" || eventType === "error") streamEnded = true;
              }
              boundary = buffer.indexOf("\n\n");
            }
          };

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, "\n");
            processFrames();
            if (streamEnded) break;
          }
          buffer += decoder.decode().replace(/\r\n/g, "\n");
          processFrames();
          if (!streamEnded) throw new Error("AI stream ended before a terminal event");
          reader.cancel().catch(() => {});
        })
        .catch((err) => {
          if (err.name === "AbortError") return;
          onError?.(err instanceof Error ? err : new Error(String(err)));
        });

      return controller;
    },

    regenerateAIMessage(id: string): Promise<SendAIMessageResponse> {
      return apiFetch<SendAIMessageResponse>(
        baseUrl,
        "POST",
        `/api/ai/conversations/${encodeURIComponent(id)}/regenerate`,
        {},
        getToken,
      );
    },

    decideAIToolCall(
      id: string,
      callId: string,
      approved: boolean,
      comment?: string,
    ): Promise<AIToolDecisionResponse> {
      return apiFetch<AIToolDecisionResponse>(
        baseUrl,
        "POST",
        `/api/ai/conversations/${encodeURIComponent(id)}/tool-calls/${encodeURIComponent(callId)}`,
        comment ? { approved, comment } : { approved },
        getToken,
      );
    },

    stopAIGeneration(id: string): Promise<StopAIGenerationResponse> {
      return apiFetch<StopAIGenerationResponse>(
        baseUrl,
        "POST",
        `/api/ai/conversations/${encodeURIComponent(id)}/stop`,
        {},
        getToken,
      );
    },

    // ─── Background tasks ─────────────────────────────────────────────────

    createAITask(data: CreateAITaskInput): Promise<CreateAITaskResponse> {
      return apiFetch<CreateAITaskResponse>(baseUrl, "POST", "/api/ai/tasks", data, getToken);
    },

    listAITasks(status?: AITaskStatus, limit?: number): Promise<AITask[]> {
      return apiFetch<AITask[]>(
        baseUrl,
        "GET",
        `/api/ai/tasks${buildQuery({ status, limit })}`,
        undefined,
        getToken,
      );
    },

    getAITask(id: string): Promise<AITask> {
      return apiFetch<AITask>(
        baseUrl,
        "GET",
        `/api/ai/tasks/${encodeURIComponent(id)}`,
        undefined,
        getToken,
      );
    },

    /** Cancels a queued task. Running tasks are not cancellable — their worker holds the lease. */
    cancelAITask(id: string): Promise<void> {
      return apiFetch<void>(
        baseUrl,
        "POST",
        `/api/ai/tasks/${encodeURIComponent(id)}/cancel`,
        {},
        getToken,
      );
    },

    /** Deletes a terminal (completed or failed) task. */
    deleteAITask(id: string): Promise<void> {
      return apiFetch<void>(
        baseUrl,
        "DELETE",
        `/api/ai/tasks/${encodeURIComponent(id)}`,
        undefined,
        getToken,
      );
    },

    /**
     * Reports runner availability and queue depth. Unlike the other task
     * methods this stays reachable when the runner is disabled, so the UI can
     * explain why rather than surfacing a bare 503.
     */
    getAITaskStatus(): Promise<AITaskStatusResponse> {
      return apiFetch<AITaskStatusResponse>(
        baseUrl,
        "GET",
        "/api/ai/tasks/status",
        undefined,
        getToken,
      );
    },

    // ─── Memory graph (v2) ────────────────────────────────────────────────

    /** Get memory store stats. */
    getAIMemoryStats(): Promise<AIMemoryStats> {
      return apiFetch<AIMemoryStats>(baseUrl, "GET", "/api/ai/memory/stats", undefined, getToken);
    },

    /** Get the full memory graph (nodes + edges from mem_root). */
    getAIMemoryGraph(): Promise<AIMemoryGraph> {
      return apiFetch<AIMemoryGraph>(baseUrl, "GET", "/api/ai/memory/graph", undefined, getToken);
    },

    /** Get a single fragment (with body) for editing. */
    getAIMemoryFragment(id: string): Promise<AIMemoryFragment> {
      return apiFetch<AIMemoryFragment>(
        baseUrl,
        "GET",
        `/api/ai/memory/fragments/${encodeURIComponent(id)}`,
        undefined,
        getToken,
      );
    },

    /** Apply a write_memory batch from the UI. */
    writeAIMemory(ops: AIMemoryWriteOp[]): Promise<AIMemoryWriteResult> {
      return apiFetch<AIMemoryWriteResult>(baseUrl, "POST", "/api/ai/memory/write", { ops }, getToken);
    },

    /** Run the maintenance job (salience decay, archive, purge, verify, reindex). */
    maintainAIMemory(): Promise<Record<string, unknown>> {
      return apiFetch<Record<string, unknown>>(baseUrl, "POST", "/api/ai/memory/maintain", {}, getToken);
    },
  };
}
