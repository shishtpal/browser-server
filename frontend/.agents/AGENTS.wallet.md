# AGENTS.wallet.md — Wallet Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers `components/wallet/`.

`WalletPage.vue` wires `wallet/composables/useWalletPage.ts` (edit modal w/ on-demand password prefill + shared-modal delete confirm) over `useWallet.ts` (list + filter + CRUD; immediate load on user change). Password reveal/copy lives in `useWalletPassword.ts` behind the shared `WalletPasswordField.vue` — never cache passwords in list state.

```
../components/wallet/
├── walletFormat.ts              # Search columns, walletInitial, isPasswordless
├── composables/useWalletPage.ts # Edit modal + reveal prefill + delete confirm
├── composables/useWallet.ts     # List + filter + CRUD (immediate load)
├── composables/useWalletPassword.ts # Per-entry reveal/copy state
├── WalletAddForm.vue, WalletSearchBar.vue, WalletEditModal.vue, WalletImport.vue
├── WalletPasswordField.vue      # •••••• mask + reveal + copy (eye/copy icons)
└── views/                       # WalletTableRow (desktop) + WalletCard (mobile)
```
