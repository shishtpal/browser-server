# AGENTS.md — Frontend (web app)

Astro + Vue + TailwindCSS web app for Browser Server. This file covers `frontend/`; the [root `AGENTS.md`](../AGENTS.md) covers the Go backend and cross-cutting concerns.

## Tech Stack

- **Astro 6** — routing via file-based pages in `src/pages/`, ships zero JS by default
- **Vue 3** (`<script setup lang="ts">`) — all interactivity lives in Vue islands
- **TailwindCSS 4** — via `@tailwindcss/vite`; global styles in `src/styles/global.css`
- **Shared workspace packages** — `@browser-server/shared-client`, `@browser-server/shared-types`, `@browser-server/shared-utils`, `@browser-server/shared-markdown` (linked from `../shared/*`)

> Note: the package is named `docs-spi` in `package.json` for historical reasons; it is the web frontend.

## Commands

Run from `frontend/` (pnpm workspace):

```bash
pnpm dev       # astro dev (local dev server, default :4321)
pnpm build     # astro build → dist/
pnpm preview   # preview the production build
```

The full release build is driven by `../scripts/build.ps1`, which runs the frontend build and copies `dist/` next to the Go binary for static serving.

```ps1
.\scripts\build.ps1 -Target Frontend
```

## Structure

```
frontend/src/
├── pages/            # Astro routes (.astro) + content (faqs.md). One per nav item.
│   ├── index.astro   # Todos (home)
│   ├── bookmarks.astro, history.astro, wallet.astro, analytics.astro, users.astro, chat.astro
│   ├── about.astro, contact.astro, 404.astro
├── layouts/Layout.astro   # Shared shell: nav, theme, header widgets
├── components/       # Vue components
│   ├── <Domain>Page.vue   # Top-level page component per domain (TodoPage, WalletPage, ChatPage, …)
│   ├── todos/, bookmarks/, history/, wallet/   # Per-domain sub-components (see below)
│   ├── chat/, quiz/, calendar/   # AI chat, exam prep, and calendar modules (see below)
│   ├── ui/           # Reusable presentational components (Button, Modal, ErrorBanner, InputField, …)
│   ├── ServerStatus.vue, ThemeToggle.vue, ApiTokenSettings.vue   # Header widgets
├── composables/      # use<Domain>() — state + data-loading logic (Vue composition API)
├── lib/
│   ├── api.ts        # App-facing API wrapper (delegates to shared client + raw fetch)
│   ├── auth.ts       # API token storage (localStorage) + authHeaders()
│   └── utils.ts      # App-specific helpers
└── types.ts          # Re-exports @browser-server/shared-types
```

### Chat module (`components/chat/`)

The AI chat UI is fully modular, split into focused sub-components and composables. The list of providers and models is read from `bs-ai-models.json` via `/api/ai/config`; behavior toggles such as `tools.allowed`, `memory`, and `skills` come from `bs-ai-config.json`. Optional MCP servers are connected only by the backend; the frontend receives sanitized status plus their runtime tool names/categories through `/api/ai/config` and must use the same selection, approval, and SSE paths as built-in tools.

```
components/chat/
├── ChatTopBar.vue          # Provider/model selects, YOLO mode toggle, mobile sidebar button
├── ChatSidebar.vue         # Desktop conversation list with search and actions
├── ChatMobileDrawer.vue    # Mobile drawer wrapping conversation list
├── ChatMessageList.vue     # Scrollable message container, empty-state suggestions, typing indicator
├── ChatBubble.vue          # Renders user, assistant (markdown), and tool messages
├── ChatInput.vue           # Auto-resizing textarea with send/stop controls
├── ChatRegenerateButton.vue# Regenerate the last assistant response
├── ChatDisabledState.vue   # Placeholder when bs-ai-config.json or bs-ai-models.json is missing
├── ChatCopyToast.vue       # Clipboard feedback toast
└── composables/
    ├── useChatConfig.ts        # AI config, provider/model state, YOLO mode persistence
    ├── useChatConversations.ts # Conversation CRUD, fork/branch, search/filter, rename/delete modals
    └── useChatMessaging.ts     # Send, stream (SSE), tool decisions, regenerate, stop
```

### Branch a conversation from any message

Every user and assistant bubble exposes a **Branch** action (git-branch icon). It calls
`forkConversation(sourceId, messageId)` (in `useChatConversations.ts`), which POSTs to
`POST /api/ai/conversations/{id}/fork` with `{ message_id }`. The backend copies every
settled message up to **and including** the selected one into a brand-new conversation
that inherits the source provider / model / profile / skills, then the UI activates the
new branch so the user can keep chatting from that point. The source conversation is left
untouched.

Markdown rendering lives in the shared package `@browser-server/shared-markdown` (`shared/browser-markdown`), consumed via `renderMarkdown` / `typesetMath` in `messages/AssistantBubble.vue`. MathJax typesetting is opt-in per assistant message (Sigma toggle in the bubble actions, off by default); `renderMarkdown` accepts `{ math: false }` to render `$…$` literally. Because the renderer's Tailwind class strings live outside the frontend root, `src/styles/global.css` includes an explicit `@source '../../../shared/browser-markdown/src'` — keep it if the package moves.

`ChatPage.vue` composes these pieces and delegates business logic to the composables, keeping the top-level component focused on wiring.

### Todos module (`components/todos/`)

`TodoPage.vue` is thin wiring; all coordination lives in `components/todos/composables/useTodoPage.ts` (view mode, editor modal, screenshot lightbox, delete confirm, kanban/list reorder) on top of `useTodos.ts` (list, filters, sort, CRUD). Presentation constants (priority/status meta, due-date predicates, sort + recurrence options) are centralized in `todoFormat.ts` — never re-declare them per component. The todo editor modal (`todos/editor/`) is owned by the todos domain and reused by the Calendar page.

```
components/todos/
├── todoFormat.ts              # PRIORITY_META, STATUS_META, due-date predicates, RRULE/SORT options
├── composables/
│   ├── useTodoPage.ts         # Page orchestration for TodoPage.vue
│   ├── useTodos.ts            # List state + filters + CRUD (immediate load on user change)
│   ├── useTodoPriority.ts, useTodoDueDate.ts, useTodoTags.ts  # Filter state only
│   ├── useTodoSort.ts, useTodoSubtasks.ts, useTodoDisplay.ts  # Card-shared display logic
│   └── useTodoListDrag.ts     # vuedraggable (mobile) + native HTML5 rows (desktop)
├── TodoActionsBar.vue         # View toggle, status tabs, filter selects, active chips
├── TodoViewToggle.vue, TodoSortBar.vue, TodoSearchBar.vue, TodoFilterSelect.vue
├── TodoStatusToggle.vue       # Round status checkbox (pending → in_progress → completed)
├── TodoPriorityBadge.vue, TodoDueDateBadge.vue, TodoTagBadges.vue, TodoMetaChips.vue
├── TodoCardActions.vue        # Pin / archive-restore / edit / delete icon actions
├── TodoSubtaskProgress.vue
├── views/                     # TodoListView (table + mobile cards), TodoTableRow, TodoCard,
│                              # TodoKanbanBoard, TodoKanbanCard, TodoGridView, TodoGridCard,
│                              # TodoSubtaskList (inline edit + reorder)
└── editor/                    # TodoEditorModal + TodoEditorForm (also used by CalendarPage)
```

### Calendar module (`components/calendar/`)

`CalendarPage.vue` is thin wiring over `composables/useCalendarPage.ts` (editor + detail modals, drag-move rescheduling, view drill-down) composing `useCalendar.ts` (view/date navigation) and `useCalendarTodos.ts` (day buckets + header stats). Day-cell drop handlers are shared via `useCalendarDragDrop.ts → useCalendarDayDrop()`; month cells reuse `CalendarDayCell.vue` (priority dots on mobile, chips on desktop). Priority/status styling is sourced from `todos/todoFormat.ts`.

```
components/calendar/
├── types.ts                   # CalendarView, CalendarDay, DateRange, CalendarStats
├── composables/
│   ├── useCalendarPage.ts     # Page orchestration for CalendarPage.vue
│   ├── useCalendar.ts         # currentDate/view/dateRange + navigation
│   ├── useCalendarTodos.ts    # Todos → visible range, per-day buckets, stats (immediate load)
│   └── useCalendarDragDrop.ts # DnD MIME helpers + useCalendarDayDrop() cell handlers
├── CalendarHeader.vue         # Prev/Today/Next + Day/Week/Month/Year switcher
├── CalendarMonthView.vue      # Grid; delegates cells to CalendarDayCell.vue
├── CalendarDayCell.vue        # One month cell (dots on mobile, chips + “+N more” on desktop)
├── CalendarWeekView.vue       # 7 columns, horizontally scrollable on mobile
├── CalendarDayView.vue        # All-day list for the selected date
├── CalendarYearView.vue       # Click-to-edit year header + 12 mini calendars
├── YearMonthCard.vue          # One mini month w/ heatmap days + count badges
├── CalendarTodoChip.vue       # Draggable todo chip
└── CalendarTodoDetail.vue     # Read-only detail modal (edit entry point)
```

### Bookmarks module (`components/bookmarks/`) & History module (`components/history/`)

Both pages are thin wiring over their page composables. Shared helpers live next to the domain; the near-identical Chrome import cards both use `ui/ImportCard.vue`. Delete confirmations go through `@browser-server/shared-modal`'s `useModal().confirmDelete` (never bare `confirm()`).

```
components/bookmarks/
├── bookmarkFormat.ts          # Search columns + parse/format helpers (host, initials, tag matching)
├── composables/
│   ├── useBookmarkPage.ts     # Page wiring: edit modal + delete confirm + tree view state
│   ├── useBookmarks.ts        # List + tag filter + search + CRUD (immediate load on user change)
│   └── useBookmarkTree.ts     # folder_path → tree edges + expansion
├── BookmarkForm.vue           # Quick-add form (local state, emits payload)
├── BookmarkSearchBar.vue      # Column select + query + flat/tree toggle + counts
├── BookmarkTagFilter.vue      # Scrollable tag pills + active-filter banner
├── BookmarkEditModal.vue, BookmarkTag.vue, BookmarkImport.vue
├── BookmarkTreeView.vue / BookmarkTreeNode.vue
└── views/                     # BookmarkFlatView = BookmarkTableRow (desktop) + BookmarkCard (mobile)

components/history/
├── composables/
│   ├── useHistoryPage.ts      # Infinite scroll (vueuse IntersectionObserver) + delete confirm
│   └── useHistory.ts          # Paged list + filter + add/delete (immediate load on user change)
├── HistoryAddForm.vue, HistorySearchBar.vue
├── HistoryTableRow.vue (desktop) / HistoryCard.vue (mobile timeline)
└── HistoryImport.vue
```

### Wallet module (`components/wallet/`) & Usage module (`components/analytics/`)

`WalletPage.vue` wires `wallet/composables/useWalletPage.ts` (edit modal w/ on-demand password prefill + shared-modal delete confirm) over `useWallet.ts` (list + filter + CRUD; immediate load on user change). Password reveal/copy lives in `useWalletPassword.ts` behind the shared `WalletPasswordField.vue` — never cache passwords in list state.

`AnalyticsPage.vue` (the "Usage" page) wires `analytics/composables/useAnalytics.ts` (summary fetch; preset/custom range; day/week/month grouping; immediate load) and renders `UsageToolbar.vue` + `DomainBreakdown.vue` + `UsageTrendChart.vue` built from `analyticsFormat.ts` constants (presets, group icons, bar palette).

Both imports use the shared `ui/ImportCard.vue`.

```
components/wallet/
├── walletFormat.ts              # Search columns, walletInitial, isPasswordless
├── composables/useWalletPage.ts # Edit modal + reveal prefill + delete confirm
├── composables/useWallet.ts     # List + filter + CRUD (immediate load)
├── composables/useWalletPassword.ts # Per-entry reveal/copy state
├── WalletAddForm.vue, WalletSearchBar.vue, WalletEditModal.vue, WalletImport.vue
├── WalletPasswordField.vue      # •••••• mask + reveal + copy (eye/copy icons)
└── views/                       # WalletTableRow (desktop) + WalletCard (mobile)

components/analytics/
├── analyticsFormat.ts           # Date presets, group options, bar palette, period labels
├── composables/useAnalytics.ts  # Summary fetch + range/grouping (immediate load)
├── UsageToolbar.vue             # Presets + custom range + day/week/month segmented control
├── DomainBreakdown.vue          # Ranked top-domain bars (favicons, durations, %)
└── UsageTrendChart.vue          # Accessible bar chart (a11y label summarizes the series)
```

### Shared modal package (`shared/browser-modal/`)

`@browser-server/shared-modal` is a self-contained imperative dialog service: `ModalHost.vue` (mounted once via `AppModalHost.vue` in the layout) renders the module-level request queue from `store.ts`; `useModal()` exposes `confirm` / `confirmDelete` / `alert`, each returning a promise. Dialogs queue, trap focus, lock body scroll, honor `persistent`, and theme via the app's `.dark` class and `bsm-*` CSS in `src/modal.css` (self-imported by the host).

### Quiz module (`components/quiz/`)

The exam-prep page (question bank, flashcards, paper generator, online exam runner) is fully modular. `QuizPage.vue` is thin wiring: it owns the user selector and delegates everything else to the module-local `composables/useQuizPage.ts`, which composes the domain composables and coordinates tabs, modals, and the exam runner. All icons come from `@lucide/vue` — never hand-write inline SVGs or use emoji glyphs here.

```
components/quiz/
├── QuizTabs.vue              # Responsive segmented tab bar (scrolls on mobile, count badges)
├── quizFormat.ts             # Single source of truth: type/difficulty metadata, label + image + date formatters
├── ui/                       # Quiz-scoped atoms
│   ├── TypeBadge.vue         # Question-type pill with icon
│   ├── DifficultyBadge.vue
│   └── TagInput.vue          # Chips input w/ datalist suggestions (form + generator)
├── dashboard/
│   ├── QuestionDashboard.vue
│   ├── DashboardBreakdownCard.vue
│   └── RecentPapersList.vue
├── questions/
│   ├── QuestionList.vue      # Header, filters, paged list
│   ├── QuestionFilters.vue   # Grid filter bar + clear-all
│   ├── QuestionTagPicker.vue # Tag filter popover (click-outside to close)
│   ├── QuestionCard.vue      # Read-only question w/ edit/delete icon actions
│   ├── QuestionPagination.vue
│   ├── QuestionModal.vue
│   └── form/
│       ├── QuestionForm.vue      # Validation + payload assembly only
│       ├── OptionsEditor.vue     # single/multiple choice option rows
│       └── ChronologyEditor.vue  # ordered items editor
├── cards/                    # Spaced-repetition flashcard session
│   ├── QuestionCards.vue     # Phase controller (idle/loading/reviewing/complete)
│   ├── ReviewSetupPanel.vue, ReviewHeader.vue, ReviewCard.vue, ReviewComplete.vue, TagSelector.vue
│   ├── CardAIAssistant.vue   # "Ask AI" panel: explain / cross-check once the answer is revealed
├── papers/
│   ├── PaperList.vue, PaperCard.vue, PaperDeleteDialog.vue, PaperDetail.vue
│   ├── PaperRunnerModal.vue  # Hosts the attempt; delegates to runner/*
│   ├── generator/            # PaperGenerator.vue + PresetBar.vue + SectionCard.vue
│   └── runner/               # Online exam: ExamTopBar, ExamQuestionCard, ExamPalette,
│                             # ExamChoiceOptions, ExamChronologyOrder, ExamInputAnswer,
│                             # ExamScoreSummary, ExamReviewList, ExamReviewItem, ExamSubmitConfirm
└── composables/              # Page-scoped (chat-style); no other page imports these
    ├── useQuizPage.ts        # Tab/modal/runner orchestration for QuizPage
    ├── useQuestions.ts       # Question bank CRUD + filters + stats + vocabulary (immediate load)
    ├── useQuizPapers.ts      # Papers CRUD + detail viewer (immediate load)
    ├── useQuestionCards.ts   # Flashcard session queue
    ├── useQuestionAI.ts      # Flashcard "Ask AI" runs (ephemeral conversations, own provider/model)
    ├── usePaperAttempt.ts    # Exam state machine (answers, flags, timer, scoring)
    └── attempts.ts           # Attempt records: localStorage persistence + shared types
```

**Ask AI on flashcards** — after an answer is revealed, `ReviewCard.vue` shows `CardAIAssistant.vue`, which offers *Explain* and *Cross-check answer* actions backed by `useQuestionAI.ts`. Prompts include the official answer/explanation (`questionExplainPrompt` / `questionCrosscheckPrompt` in `quizFormat.ts`, safe because the card is already revealed). Each run streams over an **ephemeral** AI conversation that is deleted once the answer settles, so the Chat sidebar is never polluted. The provider/model are set via a gear popover on the panel and persisted in localStorage (`bs.quiz.aiProvider` / `bs.quiz.aiModel`), independently of the Chat page's selection; the panel hides itself entirely when AI is disabled.

## Conventions

### Astro pages mount Vue islands

Pages are thin: import `Layout` and the domain's `*Page.vue`, mount it with `client:only="vue"`. Don't put logic in `.astro` files.

```astro
---
import TodoPage from '../components/TodoPage.vue'
import Layout from '../layouts/Layout.astro'
---
<Layout title="Todos">
  <main><TodoPage client:only="vue" /></main>
</Layout>
```

### Components use `<script setup lang="ts">`

All Vue components use the composition API with `<script setup>`. Keep page-level state and data loading in a `composables/use<Domain>.ts` and import it into the `*Page.vue` component.

### Composables own data + state

A composable (e.g. [`composables/useTodos.ts`](src/composables/useTodos.ts)) returns `ref`s plus async actions. The standard pattern:

- `items`, `isLoading`, `error` refs
- a `load*()` that sets `isLoading`, calls the API, and traps errors into `error`
- mutating actions (`add*`, `update*`, `remove*`) that call the API then re-`load`
- `watch` user/filter refs to reload

For complex pages like AI chat, composables can live inside the component's own directory (e.g. `components/chat/composables/`) when they are tightly coupled to a single page. The same return-refs-plus-actions pattern applies; the location just reflects scope.

### API access

- Prefer functions exported from [`lib/api.ts`](src/lib/api.ts) — they wrap the shared client (`createBrowserServerClient(API_BASE, { getToken })`).
- New endpoints belong in the **shared client** (`shared/browser-client`) first, then a thin re-export here.
- Any raw `fetch` in `lib/api.ts` MUST include the auth header: `headers: { ...authHeaders() }` (JSON) or `headers: authHeaders()` (GET/DELETE/FormData). Otherwise requests get `401`.
- `API_BASE` is `http://localhost:9191`.

### Authentication / token

- The API token is stored in `localStorage` via [`lib/auth.ts`](src/lib/auth.ts) (`getToken`/`setToken`/`clearToken`/`authHeaders`).
- [`components/ApiTokenSettings.vue`](src/components/ApiTokenSettings.vue) is the header widget for entering/clearing the token; it dispatches an `api-token-changed` event on change.
- Screenshot `<img>` URLs carry the token as a `?token=` query param (the shared client's `getScreenshotUrl` handles this) since image requests can't send headers.

### Styling

- TailwindCSS 4 utility classes; support light/dark via `dark:` variants (theme toggled by `ThemeToggle.vue`, persisted in `localStorage` under `theme`).
- Reuse `components/ui/*` primitives instead of re-styling buttons/inputs/modals.

## Adding a new page

1. Create `components/<Domain>Page.vue` (+ any `components/<domain>/` sub-components).
2. Add `composables/use<Domain>.ts` for state/data.
3. Add API methods to the shared client, then re-export via `lib/api.ts`.
4. Create `pages/<domain>.astro` mounting the page with `client:only="vue"`.
5. Add a nav entry to the `navItems` array in `layouts/Layout.astro`.
