import type browserType from 'webextension-polyfill'
import { initAutomationContentScript, initLoginProviderContentScript } from '@browser-server/extension-core/content-script'

declare const browser: typeof browserType

void initLoginProviderContentScript((message) => browser.runtime.sendMessage(message))
initAutomationContentScript()
