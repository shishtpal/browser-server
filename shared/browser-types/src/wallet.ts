export interface WalletEntry {
  id: number
  user_id: number
  username: string
  password: string
  website: string
  login_provider: string
  description: string
  created_at: string
  updated_at: string
}

export interface WalletImportResult {
  imported: number
  skipped: number
  errors: string[]
}

export interface CreateWalletInput {
  user_id: number
  website: string
  login_provider?: string
  username: string
  password: string
  description?: string
}

export interface UpdateWalletInput {
  username?: string
  password?: string
  website?: string
  login_provider?: string
  description?: string
}
