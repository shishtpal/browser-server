<script setup lang="ts">
import type { Component } from 'vue';
import {
  ArrowRight,
  Bookmark,
  Bot,
  Check,
  Clock,
  Database,
  GitFork,
  Image,
  Layers,
  ListChecks,
  Server,
  ShieldCheck,
  Sparkles,
  Wrench,
  Zap,
} from '@lucide/vue';

const capabilities: Array<{
  title: string;
  description: string;
  icon: Component;
  color: string;
  surface: string;
}> = [
  {
    title: 'Plan your day',
    description:
      'Todos, priorities, calendars, recurring work, screenshots, and reminders in one flow.',
    icon: ListChecks,
    color: 'text-indigo-600 dark:text-indigo-300',
    surface: 'bg-indigo-50 dark:bg-indigo-500/10',
  },
  {
    title: 'Keep what matters',
    description:
      'Capture bookmarks and browsing history, then search them without depending on a cloud account.',
    icon: Bookmark,
    color: 'text-cyan-600 dark:text-cyan-300',
    surface: 'bg-cyan-50 dark:bg-cyan-500/10',
  },
  {
    title: 'Work with AI',
    description:
      'Use configurable chat providers, tools, memory, image generation, voice, and durable tasks.',
    icon: Bot,
    color: 'text-violet-600 dark:text-violet-300',
    surface: 'bg-violet-50 dark:bg-violet-500/10',
  },
  {
    title: 'Study deliberately',
    description:
      'Build a question bank, generate papers, and review cards with configurable scheduling rules.',
    icon: Sparkles,
    color: 'text-amber-600 dark:text-amber-300',
    surface: 'bg-amber-50 dark:bg-amber-500/10',
  },
  {
    title: 'Create locally',
    description:
      'Generate and organize images and speech in local galleries attached to your own server.',
    icon: Image,
    color: 'text-rose-600 dark:text-rose-300',
    surface: 'bg-rose-50 dark:bg-rose-500/10',
  },
  {
    title: 'Shape the system',
    description:
      'Manage providers, models, MCP integrations, and leaf configuration from Project Settings.',
    icon: Wrench,
    color: 'text-emerald-600 dark:text-emerald-300',
    surface: 'bg-emerald-50 dark:bg-emerald-500/10',
  },
];

const principles = [
  {
    title: 'Local-first by design',
    description: 'Your databases and generated media live beside the server you operate.',
    icon: Database,
  },
  {
    title: 'Composable, not prescriptive',
    description:
      'Enable only the features, providers, tools, and integrations that fit your workflow.',
    icon: Layers,
  },
  {
    title: 'Visible control',
    description:
      'Configuration, logs, and operational state stay inspectable instead of disappearing behind a service.',
    icon: ShieldCheck,
  },
];
</script>

<template>
  <div class="mx-auto w-full max-w-[1500px] px-3 py-4 sm:px-6 sm:py-7 lg:px-10 xl:px-12">
    <section
      class="relative overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-xl shadow-slate-900/5 dark:border-white/10 dark:bg-slate-900 dark:shadow-black/20"
    >
      <div
        class="pointer-events-none absolute -top-32 -right-24 h-80 w-80 rounded-full bg-indigo-400/20 blur-3xl dark:bg-indigo-500/20"
      ></div>
      <div
        class="pointer-events-none absolute -bottom-32 -left-20 h-72 w-72 rounded-full bg-cyan-400/15 blur-3xl dark:bg-cyan-500/15"
      ></div>

      <div
        class="relative grid gap-10 p-5 sm:p-8 lg:grid-cols-[1.05fr_0.95fr] lg:items-center lg:p-12 xl:p-16"
      >
        <div>
          <span
            class="inline-flex items-center gap-2 rounded-full border border-indigo-200 bg-indigo-50 px-3 py-1.5 text-[10px] font-extrabold tracking-[0.18em] text-indigo-700 uppercase dark:border-indigo-500/20 dark:bg-indigo-500/10 dark:text-indigo-300"
          >
            <Zap class="h-3.5 w-3.5" aria-hidden="true" />
            Your browser. Your server. Your workflow.
          </span>
          <h1
            class="mt-5 max-w-3xl text-4xl leading-[1.05] font-black tracking-[-0.045em] text-slate-950 sm:text-5xl lg:text-6xl dark:text-white"
          >
            A personal control plane for the work you do online.
          </h1>
          <p
            class="mt-5 max-w-2xl text-sm leading-7 text-slate-600 sm:text-base dark:text-slate-300"
          >
            Browser Server brings capture, planning, personal data, AI workflows, and study tools
            into one self-hosted workspace—without turning your browser routine into another
            scattered set of subscriptions.
          </p>
          <div class="mt-7 flex flex-col gap-3 sm:flex-row">
            <a
              href="/"
              class="inline-flex items-center justify-center gap-2 rounded-xl bg-slate-950 px-5 py-3 text-sm font-extrabold text-white shadow-lg transition hover:-translate-y-0.5 hover:bg-indigo-600 dark:bg-white dark:text-slate-950 dark:hover:bg-indigo-300"
            >
              Open your workspace <ArrowRight class="h-4 w-4" aria-hidden="true" />
            </a>
            <a
              href="https://github.com/shishtpal/browser-server"
              target="_blank"
              rel="noreferrer"
              class="inline-flex items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white px-5 py-3 text-sm font-extrabold text-slate-700 transition hover:border-indigo-300 hover:bg-indigo-50 dark:border-white/10 dark:bg-white/5 dark:text-slate-200 dark:hover:bg-indigo-500/10"
            >
              <GitFork class="h-4 w-4" aria-hidden="true" /> View source
            </a>
          </div>
          <div class="mt-7 flex flex-wrap gap-2">
            <span
              class="rounded-full bg-slate-100 px-3 py-1.5 text-[10px] font-bold text-slate-600 dark:bg-white/10 dark:text-slate-300"
              >Self-hosted</span
            >
            <span
              class="rounded-full bg-slate-100 px-3 py-1.5 text-[10px] font-bold text-slate-600 dark:bg-white/10 dark:text-slate-300"
              >Open source</span
            >
            <span
              class="rounded-full bg-slate-100 px-3 py-1.5 text-[10px] font-bold text-slate-600 dark:bg-white/10 dark:text-slate-300"
              >Go + Vue + Astro</span
            >
            <span
              class="rounded-full bg-slate-100 px-3 py-1.5 text-[10px] font-bold text-slate-600 dark:bg-white/10 dark:text-slate-300"
              >SQLite-backed</span
            >
          </div>
        </div>

        <div
          class="relative mx-auto w-full max-w-xl"
          role="img"
          aria-label="Illustration of the Browser Server dashboard"
        >
          <div
            class="absolute inset-6 rounded-[2rem] bg-gradient-to-br from-indigo-500 to-cyan-400 opacity-30 blur-2xl"
          ></div>
          <div
            class="relative rounded-[1.75rem] border border-white/10 bg-slate-950 p-3 shadow-2xl shadow-indigo-900/30 sm:p-4"
          >
            <div class="flex items-center justify-between border-b border-white/10 px-2 pb-3">
              <div class="flex items-center gap-1.5" aria-hidden="true">
                <span class="h-2.5 w-2.5 rounded-full bg-rose-400"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-amber-300"></span>
                <span class="h-2.5 w-2.5 rounded-full bg-emerald-400"></span>
              </div>
              <span class="font-mono text-[9px] font-semibold text-slate-500">localhost:9191</span>
            </div>

            <div class="grid gap-3 pt-3 sm:grid-cols-[0.72fr_1.28fr]">
              <div class="space-y-2 rounded-2xl bg-white/5 p-3">
                <div class="mb-3 h-2 w-16 rounded-full bg-white/15"></div>
                <div
                  class="flex items-center gap-2 rounded-lg bg-indigo-500 px-2.5 py-2 text-[10px] font-bold text-white"
                >
                  <ListChecks class="h-3.5 w-3.5" aria-hidden="true" /> Todos
                </div>
                <div
                  class="flex items-center gap-2 rounded-lg px-2.5 py-2 text-[10px] font-bold text-slate-400"
                >
                  <Bookmark class="h-3.5 w-3.5" aria-hidden="true" /> Bookmarks
                </div>
                <div
                  class="flex items-center gap-2 rounded-lg px-2.5 py-2 text-[10px] font-bold text-slate-400"
                >
                  <Bot class="h-3.5 w-3.5" aria-hidden="true" /> AI chat
                </div>
                <div
                  class="flex items-center gap-2 rounded-lg px-2.5 py-2 text-[10px] font-bold text-slate-400"
                >
                  <Clock class="h-3.5 w-3.5" aria-hidden="true" /> History
                </div>
              </div>

              <div class="rounded-2xl bg-white p-3 text-slate-900 sm:p-4">
                <div class="flex items-center justify-between">
                  <div>
                    <div class="text-[9px] font-bold tracking-wider text-indigo-500 uppercase">
                      Today
                    </div>
                    <div class="mt-0.5 text-sm font-black">Personal workspace</div>
                  </div>
                  <span
                    class="grid h-8 w-8 place-items-center rounded-lg bg-indigo-50 text-indigo-600"
                  >
                    <Sparkles class="h-4 w-4" aria-hidden="true" />
                  </span>
                </div>
                <div class="mt-4 grid grid-cols-3 gap-2">
                  <div class="rounded-xl bg-slate-950 p-2.5 text-white">
                    <div class="text-base font-black">8</div>
                    <div class="text-[8px] text-slate-400">Open</div>
                  </div>
                  <div class="rounded-xl bg-emerald-50 p-2.5 text-emerald-700">
                    <div class="text-base font-black">12</div>
                    <div class="text-[8px] text-emerald-600">Done</div>
                  </div>
                  <div class="rounded-xl bg-amber-50 p-2.5 text-amber-700">
                    <div class="text-base font-black">3</div>
                    <div class="text-[8px] text-amber-600">Due</div>
                  </div>
                </div>
                <div class="mt-3 space-y-2">
                  <div class="flex items-center gap-2 rounded-xl border border-slate-100 p-2.5">
                    <span class="h-3.5 w-3.5 rounded-md border-2 border-indigo-400"></span>
                    <span class="h-2 w-3/5 rounded-full bg-slate-200"></span>
                  </div>
                  <div class="flex items-center gap-2 rounded-xl border border-slate-100 p-2.5">
                    <span
                      class="grid h-3.5 w-3.5 place-items-center rounded-md bg-emerald-500 text-white"
                    >
                      <Check class="h-2.5 w-2.5" :stroke-width="3" aria-hidden="true" />
                    </span>
                    <span class="h-2 w-4/5 rounded-full bg-slate-200"></span>
                  </div>
                  <div class="flex items-center gap-2 rounded-xl border border-slate-100 p-2.5">
                    <span class="h-3.5 w-3.5 rounded-md border-2 border-amber-400"></span>
                    <span class="h-2 w-1/2 rounded-full bg-slate-200"></span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="py-12 sm:py-16">
      <div class="max-w-3xl">
        <p
          class="text-[10px] font-extrabold tracking-[0.2em] text-indigo-600 uppercase dark:text-indigo-300"
        >
          One workspace, many jobs
        </p>
        <h2
          class="mt-2 text-2xl font-black tracking-tight text-slate-950 sm:text-3xl dark:text-white"
        >
          Built around the browser, not trapped inside it.
        </h2>
        <p class="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-400">
          The extensions make capture effortless; the server turns those captures into durable,
          searchable workflows you can inspect and adapt.
        </p>
      </div>

      <div class="mt-7 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <article
          v-for="capability in capabilities"
          :key="capability.title"
          class="group rounded-2xl border border-slate-200 bg-white p-5 shadow-sm transition hover:-translate-y-1 hover:border-indigo-200 hover:shadow-xl hover:shadow-indigo-500/5 dark:border-white/10 dark:bg-slate-900 dark:hover:border-indigo-500/30"
        >
          <span
            class="grid h-10 w-10 place-items-center rounded-xl"
            :class="[capability.surface, capability.color]"
          >
            <component :is="capability.icon" class="h-5 w-5" aria-hidden="true" />
          </span>
          <h3 class="mt-4 text-sm font-extrabold text-slate-900 dark:text-white">
            {{ capability.title }}
          </h3>
          <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-slate-400">
            {{ capability.description }}
          </p>
        </article>
      </div>
    </section>

    <section
      class="rounded-3xl border border-slate-200 bg-slate-950 p-5 text-white shadow-xl sm:p-8 lg:p-10 dark:border-white/10"
    >
      <div class="grid gap-8 lg:grid-cols-[0.8fr_1.2fr] lg:items-center">
        <div>
          <span
            class="inline-flex items-center gap-2 rounded-full bg-white/10 px-3 py-1 text-[10px] font-bold tracking-wider text-cyan-200 uppercase"
          >
            <Server class="h-3.5 w-3.5" aria-hidden="true" /> How it fits together
          </span>
          <h2 class="mt-4 text-2xl font-black tracking-tight sm:text-3xl">
            A small stack with a clear owner: you.
          </h2>
          <p class="mt-3 text-sm leading-6 text-slate-300">
            Browser clients call one Go server. The server applies token boundaries, owns the data,
            and connects optional AI providers and tools on your terms.
          </p>
        </div>
        <div class="grid gap-3 sm:grid-cols-[1fr_auto_1fr_auto_1fr] sm:items-center">
          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            <Bookmark class="h-5 w-5 text-cyan-300" aria-hidden="true" />
            <div class="mt-3 text-sm font-extrabold">Browser clients</div>
            <div class="mt-1 text-[10px] leading-4 text-slate-400">
              Web UI plus Chromium and Firefox extensions
            </div>
          </div>
          <ArrowRight class="mx-auto hidden h-4 w-4 text-slate-600 sm:block" aria-hidden="true" />
          <div class="rounded-2xl border border-indigo-400/20 bg-indigo-500/10 p-4">
            <Server class="h-5 w-5 text-indigo-300" aria-hidden="true" />
            <div class="mt-3 text-sm font-extrabold">Browser Server</div>
            <div class="mt-1 text-[10px] leading-4 text-slate-400">
              API, auth, workflows, AI runtime, and administration
            </div>
          </div>
          <ArrowRight class="mx-auto hidden h-4 w-4 text-slate-600 sm:block" aria-hidden="true" />
          <div class="rounded-2xl border border-emerald-400/20 bg-emerald-500/10 p-4">
            <Database class="h-5 w-5 text-emerald-300" aria-hidden="true" />
            <div class="mt-3 text-sm font-extrabold">Your storage</div>
            <div class="mt-1 text-[10px] leading-4 text-slate-400">
              Local SQLite databases, files, memory, and galleries
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="py-12 sm:py-16">
      <div class="grid gap-4 lg:grid-cols-3">
        <article
          v-for="principle in principles"
          :key="principle.title"
          class="rounded-2xl border border-slate-200 bg-white p-5 dark:border-white/10 dark:bg-slate-900"
        >
          <component
            :is="principle.icon"
            class="h-5 w-5 text-indigo-500 dark:text-indigo-300"
            aria-hidden="true"
          />
          <h3 class="mt-4 text-sm font-extrabold text-slate-900 dark:text-white">
            {{ principle.title }}
          </h3>
          <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-slate-400">
            {{ principle.description }}
          </p>
        </article>
      </div>
      <div
        class="mt-5 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-xs leading-5 text-amber-900 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-100"
      >
        <strong>Self-hosted still requires care.</strong> Keep the server on a trusted network,
        protect both API tokens, review configuration before enabling tools, and remember that
        wallet entries are not encrypted at rest by this application.
      </div>
    </section>

    <section
      class="mb-5 overflow-hidden rounded-3xl bg-gradient-to-r from-indigo-600 via-violet-600 to-cyan-500 p-[1px] shadow-xl shadow-indigo-500/15"
    >
      <div
        class="flex flex-col gap-5 rounded-[calc(1.5rem-1px)] bg-slate-950 px-5 py-7 text-white sm:flex-row sm:items-center sm:justify-between sm:px-8"
      >
        <div>
          <h2 class="text-xl font-black">Make the workspace yours.</h2>
          <p class="mt-1 max-w-2xl text-xs leading-5 text-slate-300">
            Start with the built-in tools, then shape providers, configuration, and browser capture
            around your own habits.
          </p>
        </div>
        <a
          href="/settings/"
          class="inline-flex shrink-0 items-center justify-center gap-2 rounded-xl bg-white px-4 py-2.5 text-xs font-extrabold text-slate-950 transition hover:-translate-y-0.5 hover:bg-cyan-100"
        >
          Open Project Settings <ArrowRight class="h-4 w-4" aria-hidden="true" />
        </a>
      </div>
    </section>
  </div>
</template>
