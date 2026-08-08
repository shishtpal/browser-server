import type { AIVoiceConfig } from '@browser-server/shared-types'
import { computed, onUnmounted, ref, watch, type Ref } from 'vue'
import { getAIVoiceConfig } from '../../../lib/api'
import { API_BASE } from '../../../lib/api/client'
import { getToken } from '../../../lib/auth'

export type VoiceState =
  | 'idle'
  | 'requesting'
  | 'listening'
  | 'transcribing'
  | 'denied'
  | 'unsupported'
  | 'error'

const PROVIDER_KEY = 'bs.ai.voice.provider'
const MODEL_KEY = 'bs.ai.voice.model'
const LANGUAGE_KEY = 'bs.ai.voice.language'

export function useVoiceTyping(open: Ref<boolean>) {
  const config = ref<AIVoiceConfig | null>(null)
  const state = ref<VoiceState>('idle')
  const error = ref('')
  const transcript = ref('')
  const provider = ref('')
  const model = ref('')
  const language = ref('')
  const elapsedSeconds = ref(0)

  let abortStart = false
  let stream: MediaStream | null = null
  let context: AudioContext | null = null
  let source: MediaStreamAudioSourceNode | null = null
  let processor: ScriptProcessorNode | null = null
  let socket: WebSocket | null = null
  let elapsedTimer: ReturnType<typeof setInterval> | null = null
  let maxTimer: ReturnType<typeof setTimeout> | null = null
  let finalizeTimer: ReturnType<typeof setTimeout> | null = null
  let connectionTimer: ReturnType<typeof setTimeout> | null = null
  let rejectConnection: ((reason?: unknown) => void) | null = null

  function clearConnectionWait() {
    if (connectionTimer) clearTimeout(connectionTimer)
    connectionTimer = null
    rejectConnection = null
  }

  let silenceSince = 0
  let speechDetected = false
  let session = 0

  const providers = computed(() =>
    Object.entries(config.value?.providers ?? {}).filter(([, item]) => item.enabled !== false),
  )
  const models = computed(() => config.value?.providers[provider.value]?.models ?? [])
  const selectedModel = computed(() => models.value.find((item) => item.id === model.value))
  const isActive = computed(
    () =>
      state.value === 'requesting' || state.value === 'listening' || state.value === 'transcribing',
  )

  function restoreSelections() {
    const cfg = config.value
    if (!cfg) return
    const savedProvider = localStorage.getItem(PROVIDER_KEY) ?? ''
    provider.value =
      cfg.providers[savedProvider]?.enabled !== false && cfg.providers[savedProvider]
        ? savedProvider
        : cfg.providers[cfg.default_provider ?? '']
          ? cfg.default_provider!
          : (providers.value[0]?.[0] ?? '')
    const providerConfig = cfg.providers[provider.value]
    const savedModel = localStorage.getItem(MODEL_KEY) ?? ''
    model.value = providerConfig?.models.some((item) => item.id === savedModel)
      ? savedModel
      : (providerConfig?.models.find((item) => item.default)?.id ??
        providerConfig?.models[0]?.id ??
        '')
    const savedLanguage = localStorage.getItem(LANGUAGE_KEY) ?? ''
    language.value = cfg.languages.some((item) => item.code === savedLanguage)
      ? savedLanguage
      : (cfg.languages[0]?.code ?? '')
  }

  async function loadConfig(expectedSession?: number): Promise<boolean> {
    try {
      const loaded = await getAIVoiceConfig()
      if (expectedSession !== undefined && (expectedSession !== session || !open.value))
        return false
      config.value = loaded
      restoreSelections()
      if (!config.value.enabled) error.value = 'Voice typing is not configured on this server.'
      return true
    } catch (err) {
      if (expectedSession !== undefined && expectedSession !== session) return false
      state.value = 'error'
      error.value =
        err instanceof Error ? err.message : 'Failed to load voice typing configuration.'
      return false
    }
  }

  async function openSession() {
    await start()
  }

  function websocketURL() {
    const url = new URL('/api/ai/voice/transcribe', API_BASE)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    url.searchParams.set('provider', provider.value)
    url.searchParams.set('model', model.value)
    url.searchParams.set('language', language.value)
    const token = getToken()
    if (token) url.searchParams.set('token', token)
    return url.toString()
  }

  async function start() {
    if (isActive.value) return

    error.value = ''
    transcript.value = ''

    if (
      !window.isSecureContext ||
      !navigator.mediaDevices?.getUserMedia ||
      !window.AudioContext ||
      !window.WebSocket
    ) {
      state.value = 'unsupported'
      error.value = 'Voice typing requires a supported browser and a secure HTTPS connection.'
      return
    }

    abortStart = false
    state.value = 'requesting'
    const currentSession = ++session

    if (!config.value) {
      const loaded = await loadConfig(currentSession)

      if (abortStart) {
        if (currentSession === session) state.value = 'idle'
        return
      }

      if (currentSession !== session || !open.value) {
        if (currentSession === session && state.value === 'requesting') state.value = 'idle'
        return
      }

      if (!loaded) return
    }

    if (!config.value?.enabled || !selectedModel.value) {
      if (currentSession === session) state.value = 'idle'
      return
    }

    if (navigator.permissions?.query) {
      try {
        const permission = await navigator.permissions.query({
          name: 'microphone' as PermissionName,
        })

        if (abortStart) {
          if (currentSession === session) state.value = 'idle'
          return
        }

        if (currentSession !== session || !open.value) {
          if (currentSession === session && state.value === 'requesting') state.value = 'idle'
          return
        }

        if (permission.state === 'denied') {
          state.value = 'denied'
          error.value =
            'Microphone access is blocked. Enable it in your browser site settings, then retry.'
          return
        }
      } catch {
        // Permission probing is optional.
      }
    }

    if (abortStart) {
      if (currentSession === session) state.value = 'idle'
      return
    }

    if (currentSession !== session || !open.value) {
      if (currentSession === session && state.value === 'requesting') state.value = 'idle'
      return
    }

    let acquiredStream: MediaStream | null = null

    try {
      acquiredStream = await navigator.mediaDevices.getUserMedia({
        audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true },
        video: false,
      })

      if (abortStart) {
        acquiredStream.getTracks().forEach((track) => track.stop())
        if (currentSession === session) state.value = 'idle'
        return
      }

      if (currentSession !== session || !open.value) {
        acquiredStream.getTracks().forEach((track) => track.stop())
        acquiredStream = null
        return
      }

      stream = acquiredStream
      await connectAndCapture(currentSession)
    } catch (err) {
      if (abortStart) {
        acquiredStream?.getTracks().forEach((track) => track.stop())
        if (currentSession === session) state.value = 'idle'
        return
      }

      if (currentSession !== session) {
        acquiredStream?.getTracks().forEach((track) => track.stop())
        return
      }

      cleanup()

      const domError = err as DOMException
      if (domError?.name === 'NotAllowedError' || domError?.name === 'SecurityError') {
        state.value = 'denied'
        error.value =
          'Microphone access was denied. Enable it in your browser site settings, then retry.'
      } else {
        state.value = 'error'
        error.value = domError?.message || 'The microphone is unavailable or already in use.'
      }
    }
  }

  async function connectAndCapture(currentSession: number) {
    clearConnectionWait()
    const currentSocket = new WebSocket(websocketURL())
    socket = currentSocket
    let opened = false
    const ready = new Promise<void>((resolve, reject) => {
      rejectConnection = reject
      connectionTimer = setTimeout(() => {
        currentSocket.close()
        reject(new Error('Timed out connecting to the voice transcription service.'))
      }, 10_000)
      currentSocket.binaryType = 'arraybuffer'
      currentSocket.onopen = () => {
        opened = true
        clearConnectionWait()
        resolve()
      }
      currentSocket.onerror = () => {
        clearConnectionWait()
        reject(new Error('Could not connect to the voice transcription service.'))
      }
      currentSocket.onclose = (event) => {
        if (!opened) {
          clearConnectionWait()
          reject(new Error('Could not connect to the voice transcription service.'))
        }
        if (currentSession !== session) return
        stopCapture()
        clearFinalizeTimer()
        if (state.value === 'transcribing' && transcript.value.trim()) state.value = 'idle'
        else if (state.value === 'listening' || state.value === 'transcribing') {
          state.value = 'error'
          error.value =
            event.reason ||
            (transcript.value ? '' : 'The transcription connection ended before text was received.')
        }
      }
      currentSocket.onmessage = (event) => {
        if (currentSession === session) handleMessage(event.data)
      }
    })
    await ready
    if (currentSession !== session || !open.value) {
      currentSocket.close()
      return
    }
    const currentContext = new AudioContext()
    await currentContext.resume()
    if (currentSession !== session || !open.value) {
      await currentContext.close()
      currentSocket.close()
      return
    }
    context = currentContext
    source = context.createMediaStreamSource(stream!)
    processor = context.createScriptProcessor(4096, 1, 1)
    processor.onaudioprocess = (event) => processAudio(event.inputBuffer.getChannelData(0))
    source.connect(processor)
    processor.connect(context.destination)
    state.value = 'listening'
    elapsedSeconds.value = 0
    speechDetected = false
    silenceSince = 0
    const started = Date.now()
    elapsedTimer = setInterval(() => {
      elapsedSeconds.value = Math.floor((Date.now() - started) / 1000)
    }, 250)
    maxTimer = setTimeout(stop, config.value!.recording.max_duration_seconds * 1000)
  }

  function processAudio(input: Float32Array) {
    if (
      state.value !== 'listening' ||
      socket?.readyState !== WebSocket.OPEN ||
      !context ||
      !selectedModel.value
    )
      return
    let sum = 0
    for (const sample of input) sum += sample * sample
    const rms = Math.sqrt(sum / input.length)
    const threshold = config.value!.recording.speech_threshold
    if (rms >= threshold) {
      speechDetected = true
      silenceSince = 0
    } else if (speechDetected) {
      silenceSince ||= Date.now()
      if (Date.now() - silenceSince >= config.value!.recording.silence_duration_ms) {
        stop()
        return
      }
    }
    const frame = toPCM16(downsample(input, context.sampleRate, selectedModel.value.sample_rate))
    if (frame.byteLength > 0) socket.send(frame)
  }

  function handleMessage(raw: unknown) {
    if (typeof raw !== 'string') return
    try {
      const message = JSON.parse(raw) as Record<string, any>
      if (message.type === 'error') {
        error.value =
          typeof message.message === 'string'
            ? message.message
            : 'The transcription provider reported an error.'
        state.value = 'error'
        cleanup()
        return
      }
      const text = findTranscript(message)
      if (text) transcript.value = mergeTranscript(transcript.value, text)
      if (message.type === 'data' && state.value === 'transcribing') {
        clearFinalizeTimer()
        state.value = 'idle'
        if (!transcript.value.trim()) error.value = 'No speech was detected. Try recording again.'
        socket?.close(1000, 'transcription complete')
      }
    } catch {
      error.value = 'The transcription service returned an invalid response.'
      state.value = 'error'
      cleanup()
    }
  }

  function stop() {
    if (state.value === 'transcribing') {
      cleanup()
      return
    }

    if (state.value !== 'listening' && state.value !== 'requesting') return

    if (state.value === 'requesting') {
      if (stream || socket) cleanup()
      else abortStart = true

      state.value = 'idle'
      return
    }

    stopCapture()
    state.value = 'transcribing'

    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'flush' }))

      finalizeTimer = setTimeout(() => {
        if (state.value !== 'transcribing') return
        state.value = 'idle'
        if (!transcript.value.trim()) error.value = 'No speech was detected. Try recording again.'
        socket?.close(1000, 'transcription complete')
      }, 10_000)
    } else {
      state.value = 'error'
      error.value = 'The transcription connection is unavailable.'
      cleanup()
    }
  }

  function stopCapture() {
    if (elapsedTimer) clearInterval(elapsedTimer)
    if (maxTimer) clearTimeout(maxTimer)
    elapsedTimer = maxTimer = null
    if (processor) {
      processor.onaudioprocess = null
      processor.disconnect()
    }
    source?.disconnect()
    stream?.getTracks().forEach((track) => track.stop())
    void context?.close().catch(() => {})
    processor = source = null
    stream = null
    context = null
  }

  function clearFinalizeTimer() {
    if (finalizeTimer) clearTimeout(finalizeTimer)
    finalizeTimer = null
  }

  function cleanup() {
    ++session
    stopCapture()
    clearFinalizeTimer()
    clearConnectionWait()
    if (socket) {
      socket.onclose = socket.onerror = socket.onmessage = null
      socket.close()
    }
    socket = null
    if (
      state.value === 'requesting' ||
      state.value === 'listening' ||
      state.value === 'transcribing'
    )
      state.value = 'idle'
  }

  function recordAgain() {
    cleanup()
    transcript.value = ''
    error.value = ''
    void start()
  }

  watch(provider, () => {
    if (!config.value) return
    const item = config.value.providers[provider.value]
    if (!item?.models.some((entry) => entry.id === model.value)) {
      model.value = item?.models.find((entry) => entry.default)?.id ?? item?.models[0]?.id ?? ''
    }
    localStorage.setItem(PROVIDER_KEY, provider.value)
  })
  watch(model, (value) => value && localStorage.setItem(MODEL_KEY, value))
  watch(language, (value) => value && localStorage.setItem(LANGUAGE_KEY, value))
  watch(open, (value) => {
    if (!value) cleanup()
  })
  onUnmounted(cleanup)

  return {
    config,
    state,
    error,
    transcript,
    provider,
    model,
    language,
    elapsedSeconds,
    providers,
    models,
    isActive,
    openSession,
    start,
    stop,
    cleanup,
    recordAgain,
  }
}

function downsample(input: Float32Array, sourceRate: number, targetRate: number): Float32Array {
  if (sourceRate === targetRate) return input
  const ratio = sourceRate / targetRate
  const output = new Float32Array(Math.floor(input.length / ratio))
  for (let i = 0; i < output.length; i++) {
    const start = Math.floor(i * ratio)
    const end = Math.min(input.length, Math.floor((i + 1) * ratio))
    let total = 0
    for (let j = start; j < end; j++) total += input[j]
    output[i] = total / Math.max(1, end - start)
  }
  return output
}

function toPCM16(input: Float32Array): ArrayBuffer {
  const buffer = new ArrayBuffer(input.length * 2)
  const view = new DataView(buffer)
  input.forEach((sample, index) => {
    const value = Math.max(-1, Math.min(1, sample))
    view.setInt16(index * 2, value < 0 ? value * 0x8000 : value * 0x7fff, true)
  })
  return buffer
}

function findTranscript(value: unknown): string {
  if (!value || typeof value !== 'object') return ''
  const item = value as Record<string, unknown>
  for (const key of ['transcript', 'text'])
    if (typeof item[key] === 'string' && item[key]) return item[key]
  for (const key of ['data', 'result', 'output']) {
    const found = findTranscript(item[key])
    if (found) return found
  }
  return ''
}

// Voice providers may send either a cumulative transcript or a new phrase after
// detecting a pause. Keep the former up to date without losing earlier phrases
// from the latter.
function mergeTranscript(existing: string, incoming: string): string {
  const current = existing.trim()
  const next = incoming.trim()
  if (!current) return next
  if (!next || current === next || current.endsWith(next)) return current
  if (next.startsWith(current)) return next
  if (current.startsWith(next)) return current

  const currentWords = current.split(/\s+/)
  const nextWords = next.split(/\s+/)
  const maximumOverlap = Math.min(currentWords.length, nextWords.length)

  for (let length = maximumOverlap; length > 0; length--) {
    if (currentWords.slice(-length).join(' ') === nextWords.slice(0, length).join(' ')) {
      return [...currentWords, ...nextWords.slice(length)].join(' ')
    }
  }

  return `${current} ${next}`
}
