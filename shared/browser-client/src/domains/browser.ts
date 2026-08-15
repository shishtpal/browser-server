import type {
  BrowserCommand,
  BrowserCreateCommandInput,
  BrowserInstance,
  BrowserRegisterInput,
  BrowserResultRequest,
  BrowserTabInfo,
  BrowserTabsInput,
} from '@browser-server/shared-types'
import { type TokenProvider, apiFetch } from '../internals'

/**
 * Browser automation domain (extension command channel). The extension uses
 * these methods to register itself, sync its tab registry, receive commands,
 * and report results. The AI tools in the CLI process use createCommand /
 * getCommand to relay through the running server.
 */
export function createBrowserMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    browserRegister(data: BrowserRegisterInput): Promise<BrowserInstance> {
      return apiFetch<BrowserInstance>(baseUrl, 'POST', '/api/browser/register', data, getToken)
    },

    browserHeartbeat(instanceId: string): Promise<{ ok: boolean }> {
      return apiFetch<{ ok: boolean }>(baseUrl, 'POST', '/api/browser/heartbeat', { instance_id: instanceId }, getToken)
    },

    browserSyncTabs(data: BrowserTabsInput): Promise<{ ok: boolean }> {
      return apiFetch<{ ok: boolean }>(baseUrl, 'POST', '/api/browser/tabs', data, getToken)
    },

    browserListInstances(): Promise<BrowserInstance[]> {
      return apiFetch<BrowserInstance[]>(baseUrl, 'GET', '/api/browser/instances', undefined, getToken)
    },

    browserListTabs(instanceId: string): Promise<BrowserTabInfo[]> {
      return apiFetch<BrowserTabInfo[]>(baseUrl, 'GET', `/api/browser/instances/${encodeURIComponent(instanceId)}/tabs`, undefined, getToken)
    },

    browserCreateCommand(data: BrowserCreateCommandInput): Promise<BrowserCommand> {
      return apiFetch<BrowserCommand>(baseUrl, 'POST', '/api/browser/cmd', data, getToken)
    },

    browserGetCommand(commandId: string): Promise<BrowserCommand> {
      return apiFetch<BrowserCommand>(baseUrl, 'GET', `/api/browser/commands/${encodeURIComponent(commandId)}`, undefined, getToken)
    },

    browserPostResult(data: BrowserResultRequest): Promise<{ ok: boolean }> {
      return apiFetch<{ ok: boolean }>(baseUrl, 'POST', '/api/browser/result', data, getToken)
    },

    browserQueue(instanceId: string): Promise<BrowserCommand[]> {
      return apiFetch<BrowserCommand[]>(baseUrl, 'GET', `/api/browser/queue?instance_id=${encodeURIComponent(instanceId)}`, undefined, getToken)
    },
  }
}
