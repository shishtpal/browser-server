import { initBackground } from '@browser-server/extension-core/background'
import { setBrowserApi } from '@browser-server/extension-core'
import { ChromeAdapter } from './adapter'

setBrowserApi(new ChromeAdapter())
initBackground()
