import { ref } from 'vue';
import { ApiError } from '@browser-server/shared-client';
import { getToken } from '../../../lib/auth';
import { generateSpeech, getAITTSConfig, getGeneratedSpeechUrl } from '../../../lib/api';

/**
 * Page-scoped TTS playback: one shared <audio> element, a single in-flight
 * generation, and a per-page availability flag so bubble actions can hide
 * themselves when bs-ai-tts.json is disabled.
 */
export function useSpeechPlayback() {
  const ttsAvailable = ref(true);
  const speakingMessageId = ref<string | null>(null);
  const speakingBusyId = ref<string | null>(null);
  let audio: HTMLAudioElement | null = null;

  async function loadTTSAvailability() {
    try {
      const cfg = await getAITTSConfig();
      ttsAvailable.value = !!cfg.enabled;
    } catch {
      ttsAvailable.value = false;
    }
  }

  function stopPlayback() {
    if (audio) {
      audio.pause();
      if (audio.src.startsWith('blob:')) URL.revokeObjectURL(audio.src);
      audio.removeAttribute('src');
      audio.load();
      audio = null;
    }
    speakingMessageId.value = null;
  }

  async function speak(messageId: string, content: string): Promise<string | null> {
    const text = content.trim();
    if (!text) return 'Nothing to read aloud';
    if (speakingBusyId.value) return null;
    if (speakingMessageId.value === messageId) {
      stopPlayback();
      return null;
    }
    stopPlayback();
    speakingBusyId.value = messageId;
    try {
      const result = await generateSpeech({ text });
      // Fetch with the auth header and play from a blob URL, so the API token
      // never lands in a query string (server/proxy access logs).
      const token = getToken();
      const res = await fetch(getGeneratedSpeechUrl(result.speech.id, false), {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) throw new ApiError(res.status, `Audio fetch failed: ${res.status}`);
      const blobUrl = URL.createObjectURL(await res.blob());
      const el = new Audio(blobUrl);
      el.addEventListener('ended', () => {
        if (speakingMessageId.value === messageId) speakingMessageId.value = null;
      });
      el.addEventListener('error', () => {
        if (speakingMessageId.value === messageId) speakingMessageId.value = null;
      });
      audio = el;
      speakingMessageId.value = messageId;
      await el.play();
      return null;
    } catch (err) {
      speakingMessageId.value = null;
      if (err instanceof ApiError && err.status === 503) {
        ttsAvailable.value = false;
        return 'TTS not configured';
      }
      return err instanceof Error ? err.message : 'Failed to generate speech';
    } finally {
      speakingBusyId.value = null;
    }
  }

  function cleanup() {
    stopPlayback();
  }

  return {
    ttsAvailable,
    speakingMessageId,
    speakingBusyId,
    loadTTSAvailability,
    speak,
    cleanup,
  };
}
