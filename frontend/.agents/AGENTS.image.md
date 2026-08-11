# AGENTS.image.md — Image Generation Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers the image generation page in `components/image/`.

`ImagePage.vue` is thin wiring over `image/composables/useImagePage.ts` (preview modal navigation, delete confirmation via the shared modal, reuse-as-source / reuse-prompt flows, prompt-library toggle). Generation state — config, form fields, gallery — lives in `useImageGeneration.ts`, which loads `bs-ai-image-models.json` via `getAIImageConfig` and derives per-model `image_sizes`, `aspect_ratios`, `max_images`, and `supports_editing`; form state is reset when the model changes so the server never receives an invalid combination. When the config is missing the page shows the disabled state explaining the server-side setup.

```
../components/image/
├── format.ts                    # Shared formatters (size labels, etc.)
├── composables/
│   ├── useImageGeneration.ts    # Config + form state + gallery CRUD (immediate load)
│   ├── useImagePage.ts          # Preview nav, delete confirm, reuse flows, prompt library
│   └── useImageZoom.ts          # Wheel/button zoom + pointer panning (1×–8×)
├── ImageComposer.vue            # Prompt/provider/model/size/aspect/count + source-image editing
├── ImageCard.vue                # Gallery card: view, use as source, reuse prompt, delete
└── ImageViewer.vue              # Preview modal with zoom/pan
```

Data comes from `lib/api/ai.ts` — `getAIImageConfig`, `listGeneratedImages`, `generateImage`, `deleteGeneratedImage`, and `getGeneratedImageUrl` (the URL carries the token as a `?token=` query param). The **Prompt Library** button opens the shared `components/prompts/PromptManager.vue` (see [`AGENTS.prompts.md`](./AGENTS.prompts.md)).
