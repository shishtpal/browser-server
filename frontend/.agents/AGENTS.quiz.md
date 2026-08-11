# AGENTS.quiz.md — Quiz / Exam-Prep Module (Frontend)

This file is part of [`AGENTS.md`](../AGENTS.md) and covers the exam-prep page in `components/quiz/`.

The exam-prep page (question bank, flashcards, paper generator, online exam runner) is fully modular. `QuizPage.vue` is thin wiring: it owns the user selector and delegates everything else to the module-local `composables/useQuizPage.ts`, which composes the domain composables and coordinates tabs, modals, and the exam runner. All icons come from `@lucide/vue` — never hand-write inline SVGs or use emoji glyphs here.

```
../components/quiz/
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

## Ask AI on flashcards

After an answer is revealed, `ReviewCard.vue` shows `CardAIAssistant.vue`, which offers _Explain_ and _Cross-check answer_ actions backed by `useQuestionAI.ts`. Prompts include the official answer/explanation (`questionExplainPrompt` / `questionCrosscheckPrompt` in `quizFormat.ts`, safe because the card is already revealed). Each run streams over an **ephemeral** AI conversation that is deleted once the answer settles, so the Chat sidebar is never polluted. The provider/model are set via a gear popover on the panel and persisted in localStorage (`bs.quiz.aiProvider` / `bs.quiz.aiModel`), independently of the Chat page's selection; the panel hides itself entirely when AI is disabled.
