# AI Video Generation Module

The text-to-video feature lives in `internal/ai/videos/` and mirrors the
`internal/ai/images` module: it is self-contained (config, store, provider
clients, service, tools), gated by a sibling config file, and wired into the
AI bootstrap the same way as TTS/images.

## Enabling

Place `bs-ai-video-models.json` next to the server binary (the same directory
anchor used by `bs-ai-config.json`). When the file is missing or
`"enabled": false`, the feature is a no-op: no gallery database is created, the
API routes answer `503 videos_disabled`, and the `generate_video` AI tool
reports `"video generation is disabled"`. A *parseable* but semantically
invalid enabled file (e.g. a bad provider key or empty `env:` API key) surfaces
as a warning at startup instead of taking the whole AI bootstrap down; the
feature stays disabled until the config reloads cleanly.

## Config shape

```jsonc
{
  "enabled": true,
  "default_provider": "agnes",
  // Optional; override the default storage locations. Relative paths are
  // resolved against the configured data directory (.data/ by default), so
  // DATA_PATH is honored.
  "db_path": "ai-videos.db",
  "video_dir": "ai-videos",
  "providers": {
    // Provider types: "agnes_video" (Agnes Video V2.0) or "openrouter_video"
    // (OpenRouter /api/v1/videos — google/veo-3.1-lite,
    // bytedance/seedance-2.0-mini, bytedance/seedance-2.0-fast).
    "agnes": {
      "type": "agnes_video",
      "base_url": "https://apihub.agnes-ai.com", // normalized to the origin
      "api_key": "env:AGNES_API_KEY", // env: prefix resolved at load time
      "request_timeout_seconds": 900, // long: video rendering takes minutes
      "models": [
        {
          "id": "agnes-video-v2.0",
          "label": "Agnes Video V2.0",
          "default": true,
          // The frontend renders one field per parameter spec, so provider
          // specific knobs are a config change, not a UI change.
          "parameters": [
            {
              "key": "prompt",
              "type": "text",
              "label": "Prompt",
              "group": "Core",
              "required": true
            },
            {
              "key": "mode",
              "type": "select",
              "label": "Mode",
              "group": "Core",
              "options": [
                "ti2vid",
                "keyframes"
              ],
              "help": "text-to-video or keyframe-to-video"
            },
            {
              "key": "width",
              "type": "number",
              "group": "Size",
              "min": 64,
              "max": 1920,
              "step": 8,
              "default": 1152
            },
            {
              "key": "height",
              "type": "number",
              "group": "Size",
              "min": 64,
              "max": 1080,
              "step": 8,
              "default": 768
            },
            {
              "key": "num_frames",
              "type": "number",
              "group": "Timing",
              "min": 9,
              "max": 241,
              "step": 8,
              "default": 121,
              "help": "must satisfy 8n+1"
            },
            {
              "key": "frame_rate",
              "type": "number",
              "group": "Timing",
              "min": 8,
              "max": 30,
              "default": 25
            },
            {
              "key": "num_inference_steps",
              "type": "number",
              "group": "Quality",
              "min": 1,
              "max": 100,
              "default": 50
            },
            {
              "key": "seed",
              "type": "number",
              "group": "Quality",
              "help": "omit for a random seed"
            },
            {
              "key": "negative_prompt",
              "type": "text",
              "group": "Quality"
            },
            {
              "key": "image",
              "type": "image_urls",
              "label": "Image (I2V)",
              "group": "Inputs",
              "help": "single public image URL"
            },
            {
              "key": "extra_body.image",
              "type": "image_urls",
              "label": "Keyframes",
              "group": "Inputs",
              "help": "array of public image URLs; triggers keyframes mode"
            }
          ]
        }
      ]
    }
  }
}
```

Parameter spec fields: `key`, `label`, `type` (`text`, `textarea`, `number`,
`select`, `boolean`, `image_urls`), `group`, `default`, `required`, `min`,
`max`, `step`, `options` (for `select`), `help`. Every model **must** declare
`parameters`; the loader rejects models without them.

`db_path` / `video_dir` may be overridden at runtime by the
`AI_VIDEO_DB_PATH` / `AI_VIDEO_VIDEO_DIR` environment variables, which take
precedence over the config values.

## Provider model

`internal/ai/videos/provider.go` defines the `providerImpl` interface
(`Create`, `Poll`) and `newProviderImpl`, which maps a provider `type` string
to its implementation. Adding a new backend means implementing those two
methods plus a provider-specific `validate<Name>Constraints` helper in
`generate.go` and registering the type in `newProviderImpl` — no changes to
the store, service, API, or UI.

Vendors that must fetch the finished video through the provider API (with the
same Authorization header) instead of a plain GET additionally implement the
optional `contentFetcher` interface (`Fetch(ctx, Provider, pollResult)`). The
service falls back to a plain GET of `pollResult.VideoURL` when a provider does
not implement it. Providers that do not report `duration`/`size` in their job
response fall back to the values the request asked for (stored in the record's
`params`).

### Agnes

`agnes.go` implements the Agnes Video V2.0 async protocol:

- `POST {origin}/v1/videos` starts a generation. The JSON response yields a
  `video_id` (recommended for retrieval) and a `task_id`. `base_url` is
  normalized to the origin — a trailing `/v1` is stripped automatically.
- Poll `GET {origin}/agnesapi?video_id=<id>` (plus optional `model_name`). The
  parser reads `status` (`queued`/`in_progress`/`completed`/`failed`),
  `progress`, `metadata.url` / `remixed_from_video_id` / `url`, and `seconds`
  (a JSON number or string, parsed leniently).
- On `completed`, the service downloads the MP4 from the result URL into
  `video_dir/<id>.mp4` and marks the record completed.

`createPayload` is provider-agnostic: it forwards known numeric/text fields
(`width`, `height`, `num_frames`, `frame_rate`, `num_inference_steps`, `seed`,
`negative_prompt`), a `prompt`, and the `image` (single URL for image-to-video)
or `extra_body.image` (array for keyframes). Only Agnes-recognized mode values
(`ti2vid`, `keyframes`) are forwarded as a top-level `mode`; other values
(e.g. `image_to_video`) are implied by the image field. `num_frames` must
satisfy `8n+1` and is validated against the model's configured min/max spec
clamped by the documented absolute ceiling (441), so spec bounds are the
single source of truth. `validateAgnesConstraints` also auto-promotes the
request `mode` to `keyframes` when keyframe URLs are supplied without it —
Agnes requires the pairing, and silently ignoring `extra_body.image` while the
mode stays on its default produces an ambiguous payload. Keyframes always
require at least two image URLs.

### Service

- `Close` waits for any in-flight advance (tracked in `s.wg`), including its
  result download, before releasing the gallery DB. A hot reload therefore
  blocks for the full provider `request_timeout_seconds` when a video download
  is in progress.
- `POST /api/ai/videos` and the `generate_video` tool return a queued record;
  rendering typically takes minutes. The tool contract tells the calling
  model to poll `GET /api/ai/videos` and never claim the `url` is playable
  before the record's status flips to `completed`.

### OpenRouter (`openrouter_video`)

`openrouter.go` implements OpenRouter's asynchronous video API for
`google/veo-3.1-lite`, `bytedance/seedance-2.0-mini`, and
`bytedance/seedance-2.0-fast`:

- `POST {origin}/api/v1/videos` submits a generation and returns a job `id`.
  The payload maps from the normalized params: `prompt`, `duration`/`seed`
  (coerced to integers — select params surface as strings like `"6"`),
  `size`/`resolution`/`aspect_ratio`/`callback_url`, `generate_audio`,
  `frame_images` (wrapped as `image_url` refs with `frame_type`; at most two
  URLs — first image pins `first_frame`, second pins `last_frame`), and
  `input_references`. `base_url` defaults to `https://openrouter.ai` and is
  kept at the origin (no `/v1` stripping).
- Poll `GET {origin}/api/v1/videos/{id}` with the same Bearer header while
  `status` is `pending`/`in_progress`; `completed` resolves the content URL,
  and `failed`/`cancelled`/`expired` are terminal failures. The API exposes no
  numeric progress, so in-flight records report `progress: 0`.
- On completion the video is fetched from
  `GET {origin}/api/v1/videos/{id}/content` via the `contentFetcher` interface
  (Bearer header included), then written to `video_dir/<id>.mp4`.

## Service & polling

`service.go` owns the gallery SQLite DB (`ai_videos` table), the HTTP client,
and a background poller that runs every 5s (and is kicked immediately on
submit). The poller advances every non-terminal record: it looks up the
provider implementation by `provider`, polls, and on completion downloads the
file. Each task gets its own context bounded by the provider's
`request_timeout_seconds` so large downloads are not cut short by the shared
poll tick. Transient poll errors are retried (logged, status left unchanged);
an explicit `failed` status from the provider (or a task that disappears)
marks the record failed so it is never polled forever.

`ServiceHolder` (in `bootstrap`) lets the admin Project Settings editor hot-
reload the config: `Swap` closes the old service (stops its poller) and starts
a new one, which re-adopts pending records from the DB.

## Routes (`/api/ai/videos/*`)

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/config` | `{enabled, default_provider, providers: {name: {models: [...]}}}` (no secrets) |
| GET | `/` | list gallery records, newest first (`?limit=`, 1..100, default 50) |
| POST | `/` | `{prompt, provider?, model?, params?}` → `queued` `GeneratedVideo` + kicks poller |
| GET | `/file/{id}` | streams the MP4 (token via `?token=`) |
| DELETE | `/{id}` | deletes DB row + file |

## AI tool

`generate_video` is registered in `internal/ai/bootstrap/bootstrap.go` with the
schema in `generate_video.json`. It returns `{video, url, note}` for the
queued gallery record; the `note` and the tool's description make the async
lifecycle explicit (renders take minutes; the URL only plays once `status`
flips to `completed`; progress is polled from `GET /api/ai/videos`). The
tool is gated by the same config: missing/disabled → `"video generation is
disabled"`. To make it available to the model it must also be listed in
`bs-ai-config.json` → `tools.allowed[]` (and is part of the `knownToolNames`
allowlist in `internal/ai/config/validate.go`).

## Frontend

- Route: `/video` (`frontend/src/pages/video.astro`, client-only island
  rendering `frontend/src/components/VideoPage.vue`).
- `shared/browser-types` — `AIVideoConfig`, `AIVideoModel`, `VideoParamSpec`,
  `GeneratedVideo`, `VideoStatus`, `GenerateVideoInput`, `GenerateVideoResponse`.
- `shared/browser-client` — typed `getAIVideoConfig`, `listGeneratedVideos`,
  `generateVideo`, `deleteGeneratedVideo`, `getGeneratedVideoUrl`.
- `frontend/src/components/video/*` — `ParamField.vue` (renders fields from the
  config spec), `VideoComposer.vue`, `VideoCard.vue`, `VideoViewer.vue`
  (fullscreen viewer), `format.ts`, and the `useVideoGeneration` /
  `useVideoPage` composables (submit + 3s polling while tasks are in flight).
