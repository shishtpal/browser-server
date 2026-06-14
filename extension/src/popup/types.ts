export type PanelKey = 'history' | 'todos' | 'wallet' | 'bookmarks'

export type PanelState = 'loading' | 'ready' | 'error'

export interface PanelStatus {
  count: number
  state: PanelState
}
