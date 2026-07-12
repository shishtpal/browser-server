import { initBackground } from '@browser-server/extension-core/background'
import { setBrowserApi } from '@browser-server/extension-core'
import { FirefoxAdapter } from './adapter'

setBrowserApi(new FirefoxAdapter())
initBackground()
