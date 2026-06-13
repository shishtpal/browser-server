# Browser Server Extension

This extension is being migrated to a Vite + TypeScript + Tailwind CSS v4 toolchain.

## Commands

```bash
vp install
vp run build
```

## Load In Browser

After building, load the unpacked extension from `extension/`.
The root `manifest.json` points Chrome/Edge to the built files under `dist/`.

## Current Scope

- Vite multi-entry build for popup, options, and background
- Shared API client under `shared/browser-client`
- Tailwind v4-based popup and options UI
- MV3 service worker bundled from TypeScript
