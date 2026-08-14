<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue';
import {
  ArrowRight,
  BookOpen,
  Bug,
  Check,
  ExternalLink,
  GitFork,
  Lightbulb,
  MessageSquareText,
  Send,
  ShieldAlert,
  Sparkles,
} from '@lucide/vue';

type ContactTopic = 'bug' | 'feature' | 'question';

const topic = ref<ContactTopic>('bug');
const subject = ref('');
const details = ref('');
const environment = ref('');
const opened = ref(false);
let openedTimer: number | undefined;

const topicMeta: Record<ContactTopic, { label: string; prefix: string; labels: string }> = {
  bug: { label: 'Bug report', prefix: '[Bug]', labels: 'bug' },
  feature: { label: 'Feature request', prefix: '[Feature]', labels: 'enhancement' },
  question: { label: 'Question', prefix: '[Question]', labels: '' },
};
const topicOptions: ContactTopic[] = ['bug', 'feature', 'question'];

const issueURL = computed(() => {
  const selected = topicMeta[topic.value];
  const params = new URLSearchParams();
  params.set('title', `${selected.prefix} ${subject.value.trim()}`.trim());
  params.set(
    'body',
    [
      '## What would you like to share?',
      details.value.trim() || '_Add the details here._',
      '',
      '## Environment or context',
      environment.value.trim() || '_Not provided._',
      '',
      '---',
      'Created from the Browser Server contact page.',
    ].join('\n'),
  );
  if (selected.labels) params.set('labels', selected.labels);
  return `https://github.com/shishtpal/browser-server/issues/new?${params.toString()}`;
});

function openIssue() {
  window.open(issueURL.value, '_blank', 'noopener,noreferrer');
  opened.value = true;
  if (openedTimer) window.clearTimeout(openedTimer);
  openedTimer = window.setTimeout(() => (opened.value = false), 4000);
}

onBeforeUnmount(() => {
  if (openedTimer) window.clearTimeout(openedTimer);
});
</script>

<template>
  <div class="mx-auto w-full max-w-[1400px] px-3 py-4 sm:px-6 sm:py-7 lg:px-10 xl:px-12">
    <section
      class="relative overflow-hidden rounded-3xl border border-slate-200 bg-slate-950 text-white shadow-xl shadow-slate-900/15 dark:border-white/10"
    >
      <div
        class="pointer-events-none absolute -top-32 right-0 h-80 w-80 rounded-full bg-cyan-400/20 blur-3xl"
      ></div>
      <div
        class="pointer-events-none absolute -bottom-28 left-1/4 h-72 w-72 rounded-full bg-violet-500/20 blur-3xl"
      ></div>
      <div class="relative grid gap-8 p-5 sm:p-8 lg:grid-cols-[1fr_0.8fr] lg:items-center lg:p-12">
        <div>
          <span
            class="inline-flex items-center gap-2 rounded-full border border-cyan-300/20 bg-cyan-300/10 px-3 py-1.5 text-[10px] font-extrabold tracking-[0.18em] text-cyan-200 uppercase"
          >
            <MessageSquareText class="h-3.5 w-3.5" aria-hidden="true" />
            Contact the project
          </span>
          <h1
            class="mt-5 max-w-3xl text-4xl leading-[1.05] font-black tracking-[-0.04em] sm:text-5xl lg:text-6xl"
          >
            Good software gets better through clear conversations.
          </h1>
          <p class="mt-5 max-w-2xl text-sm leading-7 text-slate-300 sm:text-base">
            Found a rough edge, have an idea, or need help understanding the project? Start with the
            right channel and include enough context for someone else to reproduce or discuss it.
          </p>
          <div class="mt-7 flex flex-wrap gap-2">
            <span class="rounded-full bg-white/10 px-3 py-1.5 text-[10px] font-bold text-slate-200"
              >Public issue tracking</span
            >
            <span class="rounded-full bg-white/10 px-3 py-1.5 text-[10px] font-bold text-slate-200"
              >Open-source collaboration</span
            >
            <span class="rounded-full bg-white/10 px-3 py-1.5 text-[10px] font-bold text-slate-200"
              >No hidden contact service</span
            >
          </div>
        </div>

        <div class="rounded-2xl border border-white/10 bg-white/5 p-4 backdrop-blur sm:p-5">
          <div class="flex items-start gap-3">
            <span
              class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-amber-400/15 text-amber-300"
            >
              <ShieldAlert class="h-5 w-5" aria-hidden="true" />
            </span>
            <div>
              <h2 class="text-sm font-extrabold">Share safely</h2>
              <p class="mt-1 text-xs leading-5 text-slate-400">
                Issues are public. Never paste API tokens, provider keys, wallet contents, private
                history, or unredacted configuration.
              </p>
            </div>
          </div>
          <div class="mt-4 space-y-2 border-t border-white/10 pt-4 text-[11px] text-slate-300">
            <div class="flex items-center gap-2">
              <Check class="h-3.5 w-3.5 text-emerald-300" aria-hidden="true" /> Remove secrets from
              logs
            </div>
            <div class="flex items-center gap-2">
              <Check class="h-3.5 w-3.5 text-emerald-300" aria-hidden="true" /> Describe expected
              and actual behavior
            </div>
            <div class="flex items-center gap-2">
              <Check class="h-3.5 w-3.5 text-emerald-300" aria-hidden="true" /> Include OS and
              browser details when relevant
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="py-8 sm:py-10">
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <a
          href="https://github.com/shishtpal/browser-server/issues"
          target="_blank"
          rel="noreferrer"
          class="group rounded-2xl border border-slate-200 bg-white p-4 shadow-sm transition hover:-translate-y-1 hover:border-rose-300 hover:shadow-xl dark:border-white/10 dark:bg-slate-900 dark:hover:border-rose-500/30"
        >
          <span
            class="grid h-9 w-9 place-items-center rounded-xl bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-300"
          >
            <Bug class="h-4 w-4" aria-hidden="true" />
          </span>
          <div class="mt-4 flex items-center justify-between gap-2">
            <h2 class="text-sm font-extrabold text-slate-900 dark:text-white">Report a bug</h2>
            <ExternalLink
              class="h-3.5 w-3.5 text-slate-400 transition group-hover:text-rose-500"
              aria-hidden="true"
            />
          </div>
          <p class="mt-1.5 text-xs leading-5 text-slate-500 dark:text-slate-400">
            Search existing reports or document a reproducible problem.
          </p>
        </a>

        <a
          href="https://github.com/shishtpal/browser-server/issues?q=is%3Aissue+label%3Aenhancement"
          target="_blank"
          rel="noreferrer"
          class="group rounded-2xl border border-slate-200 bg-white p-4 shadow-sm transition hover:-translate-y-1 hover:border-amber-300 hover:shadow-xl dark:border-white/10 dark:bg-slate-900 dark:hover:border-amber-500/30"
        >
          <span
            class="grid h-9 w-9 place-items-center rounded-xl bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300"
          >
            <Lightbulb class="h-4 w-4" aria-hidden="true" />
          </span>
          <div class="mt-4 flex items-center justify-between gap-2">
            <h2 class="text-sm font-extrabold text-slate-900 dark:text-white">Suggest an idea</h2>
            <ExternalLink
              class="h-3.5 w-3.5 text-slate-400 transition group-hover:text-amber-500"
              aria-hidden="true"
            />
          </div>
          <p class="mt-1.5 text-xs leading-5 text-slate-500 dark:text-slate-400">
            Explain the workflow problem before proposing the interface.
          </p>
        </a>

        <a
          href="https://github.com/shishtpal/browser-server#readme"
          target="_blank"
          rel="noreferrer"
          class="group rounded-2xl border border-slate-200 bg-white p-4 shadow-sm transition hover:-translate-y-1 hover:border-cyan-300 hover:shadow-xl dark:border-white/10 dark:bg-slate-900 dark:hover:border-cyan-500/30"
        >
          <span
            class="grid h-9 w-9 place-items-center rounded-xl bg-cyan-50 text-cyan-600 dark:bg-cyan-500/10 dark:text-cyan-300"
          >
            <BookOpen class="h-4 w-4" aria-hidden="true" />
          </span>
          <div class="mt-4 flex items-center justify-between gap-2">
            <h2 class="text-sm font-extrabold text-slate-900 dark:text-white">Read the docs</h2>
            <ExternalLink
              class="h-3.5 w-3.5 text-slate-400 transition group-hover:text-cyan-500"
              aria-hidden="true"
            />
          </div>
          <p class="mt-1.5 text-xs leading-5 text-slate-500 dark:text-slate-400">
            Review setup, configuration, and development guidance first.
          </p>
        </a>

        <a
          href="https://github.com/shishtpal/browser-server"
          target="_blank"
          rel="noreferrer"
          class="group rounded-2xl border border-slate-200 bg-white p-4 shadow-sm transition hover:-translate-y-1 hover:border-indigo-300 hover:shadow-xl dark:border-white/10 dark:bg-slate-900 dark:hover:border-indigo-500/30"
        >
          <span
            class="grid h-9 w-9 place-items-center rounded-xl bg-indigo-50 text-indigo-600 dark:bg-indigo-500/10 dark:text-indigo-300"
          >
            <GitFork class="h-4 w-4" aria-hidden="true" />
          </span>
          <div class="mt-4 flex items-center justify-between gap-2">
            <h2 class="text-sm font-extrabold text-slate-900 dark:text-white">Explore source</h2>
            <ExternalLink
              class="h-3.5 w-3.5 text-slate-400 transition group-hover:text-indigo-500"
              aria-hidden="true"
            />
          </div>
          <p class="mt-1.5 text-xs leading-5 text-slate-500 dark:text-slate-400">
            Inspect the implementation or prepare a focused contribution.
          </p>
        </a>
      </div>
    </section>

    <section class="grid gap-5 pb-8 lg:grid-cols-[1.1fr_0.9fr] lg:items-start">
      <form
        class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-lg shadow-slate-900/5 dark:border-white/10 dark:bg-slate-900"
        @submit.prevent="openIssue"
      >
        <div class="border-b border-slate-100 p-5 sm:p-6 dark:border-white/10">
          <span
            class="inline-flex items-center gap-2 text-[10px] font-extrabold tracking-[0.16em] text-violet-600 uppercase dark:text-violet-300"
          >
            <Sparkles class="h-3.5 w-3.5" aria-hidden="true" /> Issue composer
          </span>
          <h2 class="mt-2 text-xl font-black text-slate-950 dark:text-white">
            Prepare a clear GitHub issue
          </h2>
          <p class="mt-1 text-xs leading-5 text-slate-500 dark:text-slate-400">
            This form does not transmit data to Browser Server. It opens GitHub with a prefilled
            draft for you to review and submit.
          </p>
        </div>

        <div class="space-y-4 p-5 sm:p-6">
          <fieldset>
            <legend class="mb-2 text-xs font-bold text-slate-600 dark:text-slate-300">
              What is this about?
            </legend>
            <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <label
                v-for="option in topicOptions"
                :key="option"
                class="cursor-pointer rounded-xl border p-3 text-xs font-bold transition"
                :class="
                  topic === option
                    ? 'border-violet-400 bg-violet-50 text-violet-700 ring-2 ring-violet-100 dark:border-violet-400/60 dark:bg-violet-500/10 dark:text-violet-200 dark:ring-violet-500/10'
                    : 'border-slate-200 text-slate-600 hover:bg-slate-50 dark:border-white/10 dark:text-slate-300 dark:hover:bg-white/5'
                "
              >
                <input v-model="topic" type="radio" name="topic" :value="option" class="sr-only" />
                {{ topicMeta[option].label }}
              </label>
            </div>
          </fieldset>

          <div>
            <label
              for="contact-subject"
              class="mb-1.5 block text-xs font-bold text-slate-600 dark:text-slate-300"
              >Short summary</label
            >
            <input
              id="contact-subject"
              v-model="subject"
              required
              maxlength="160"
              placeholder="What should someone understand at a glance?"
              class="w-full rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-3 text-sm font-semibold text-slate-800 transition outline-none placeholder:text-slate-400 focus:border-violet-400 focus:ring-4 focus:ring-violet-100 dark:border-white/10 dark:bg-slate-950 dark:text-slate-100 dark:focus:ring-violet-500/10"
            />
          </div>

          <div>
            <label
              for="contact-details"
              class="mb-1.5 block text-xs font-bold text-slate-600 dark:text-slate-300"
              >Details</label
            >
            <textarea
              id="contact-details"
              v-model="details"
              required
              rows="7"
              placeholder="Describe the goal, actual behavior, expected behavior, and steps to reproduce."
              class="w-full resize-y rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-3 text-sm leading-6 text-slate-800 transition outline-none placeholder:text-slate-400 focus:border-violet-400 focus:ring-4 focus:ring-violet-100 dark:border-white/10 dark:bg-slate-950 dark:text-slate-100 dark:focus:ring-violet-500/10"
            ></textarea>
          </div>

          <div>
            <label
              for="contact-environment"
              class="mb-1.5 block text-xs font-bold text-slate-600 dark:text-slate-300"
            >
              Environment <span class="font-medium text-slate-400">(optional)</span>
            </label>
            <input
              id="contact-environment"
              v-model="environment"
              placeholder="Windows 11, Chrome 140, server commit…"
              class="w-full rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-3 text-sm font-semibold text-slate-800 transition outline-none placeholder:text-slate-400 focus:border-violet-400 focus:ring-4 focus:ring-violet-100 dark:border-white/10 dark:bg-slate-950 dark:text-slate-100 dark:focus:ring-violet-500/10"
            />
          </div>

          <button
            type="submit"
            class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-violet-600 to-indigo-600 px-5 py-3 text-sm font-extrabold text-white shadow-lg shadow-violet-500/20 transition hover:-translate-y-0.5 hover:shadow-xl focus:ring-4 focus:ring-violet-200 focus:outline-none dark:focus:ring-violet-500/20"
          >
            <Send class="h-4 w-4" aria-hidden="true" /> Review draft on GitHub
          </button>

          <p
            v-if="opened"
            class="flex items-center gap-2 rounded-xl bg-emerald-50 px-3 py-2.5 text-xs font-bold text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300"
            role="status"
          >
            <Check class="h-4 w-4" aria-hidden="true" /> GitHub draft requested. If no tab appeared,
            allow popups and try again.
          </p>
        </div>
      </form>

      <aside class="space-y-4">
        <div
          class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6 dark:border-white/10 dark:bg-slate-900"
        >
          <h2 class="text-base font-black text-slate-950 dark:text-white">
            A useful report answers four questions
          </h2>
          <ol class="mt-4 space-y-4">
            <li
              v-for="(item, index) in [
                'What were you trying to do?',
                'What actually happened?',
                'What did you expect instead?',
                'How can someone reproduce it?',
              ]"
              :key="item"
              class="flex gap-3"
            >
              <span
                class="grid h-7 w-7 shrink-0 place-items-center rounded-lg bg-slate-950 text-[10px] font-black text-white dark:bg-white dark:text-slate-950"
                >{{ index + 1 }}</span
              >
              <span class="pt-1 text-xs leading-5 text-slate-600 dark:text-slate-300">{{
                item
              }}</span>
            </li>
          </ol>
        </div>

        <div
          class="rounded-3xl border border-indigo-200 bg-indigo-50 p-5 sm:p-6 dark:border-indigo-500/20 dark:bg-indigo-500/10"
        >
          <GitFork class="h-5 w-5 text-indigo-600 dark:text-indigo-300" aria-hidden="true" />
          <h2 class="mt-4 text-base font-black text-indigo-950 dark:text-white">
            Prefer to contribute directly?
          </h2>
          <p class="mt-2 text-xs leading-5 text-indigo-800/75 dark:text-indigo-200/75">
            Fork the repository, create a focused branch, keep secrets and generated output out of
            commits, and describe the checks you ran.
          </p>
          <a
            href="https://github.com/shishtpal/browser-server#contributing-with-git"
            target="_blank"
            rel="noreferrer"
            class="mt-4 inline-flex items-center gap-2 text-xs font-extrabold text-indigo-700 hover:text-indigo-500 dark:text-indigo-300"
          >
            Read the contribution guide <ArrowRight class="h-3.5 w-3.5" aria-hidden="true" />
          </a>
        </div>
      </aside>
    </section>
  </div>
</template>
