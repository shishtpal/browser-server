import { onMounted, onUnmounted } from 'vue';

/**
 * Syncs the active conversation with the URL (`/chat/<id>`), handles
 * popstate (browser back/forward), and parses the initial location.
 */
export function useChatRouting(onSelectFromURL: (id: string) => Promise<void>) {
  const PREFIX = '/chat/';

  function conversationIdFromLocation(): string | null {
    if (!window.location.pathname.startsWith(PREFIX)) return null;
    const encodedID = window.location.pathname.slice(PREFIX.length);
    if (!encodedID || encodedID.includes('/')) return null;
    try {
      return decodeURIComponent(encodedID);
    } catch {
      return null;
    }
  }

  function updateConversationURL(id: string | null) {
    const pathname = id ? `${PREFIX}${encodeURIComponent(id)}` : PREFIX;
    if (window.location.pathname === pathname) return;
    window.history.pushState({}, '', `${pathname}${window.location.search}${window.location.hash}`);
  }

  async function onPopState() {
    const id = conversationIdFromLocation();
    if (!id) return;
    await onSelectFromURL(id);
  }

  onMounted(() => window.addEventListener('popstate', onPopState));
  onUnmounted(() => window.removeEventListener('popstate', onPopState));

  return { conversationIdFromLocation, updateConversationURL };
}
