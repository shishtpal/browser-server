import { initLoginProviderContentScript } from '@browser-server/extension-core/content-script'
import { initAutomationContentScript } from '@browser-server/extension-core/content-script'

void initLoginProviderContentScript((message) => chrome.runtime.sendMessage(message))
initAutomationContentScript()
