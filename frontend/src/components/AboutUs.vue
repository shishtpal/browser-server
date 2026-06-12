<template>
  <div class="mx-auto max-w-5xl px-4 py-10 sm:px-6 lg:px-8">
    <section class="overflow-hidden rounded-[2rem] border border-gray-200/80 bg-white/90 p-6 shadow-2xl shadow-gray-900/10 backdrop-blur-xl transition-colors dark:border-white/10 dark:bg-slate-800/90 dark:shadow-slate-950/20 sm:p-10">
      <div class="grid gap-8 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
        <div>
          <p class="mb-3 inline-flex rounded-full bg-indigo-50 px-3 py-1 text-xs font-bold uppercase tracking-[0.2em] text-indigo-600 transition-colors dark:bg-indigo-900/20 dark:text-indigo-400">About Browser Server</p>
          <h1 class="text-4xl font-black tracking-tight text-slate-900 transition-colors dark:text-white sm:text-6xl">A modern personal data hub for your browser workflow.</h1>
          <div class="mt-6 space-y-4 text-base leading-7 text-slate-600 transition-colors dark:text-slate-400">
            <p>Browser Server helps you manage todos, bookmarks, browsing history, users, and secure wallet entries from one clean interface.</p>
            <p>It is built for everyday productivity: fast CRUD actions, responsive layouts, helpful filters, and a calm visual design that works on phones, tablets, and desktops.</p>
            <p>Our goal is simple: keep your personal information organized without adding clutter.</p>
          </div>
        </div>
        <div class="rounded-[2rem] bg-gradient-to-br from-indigo-600 via-violet-600 to-cyan-500 p-1 shadow-2xl shadow-indigo-500/25">
          <div class="rounded-[1.75rem] bg-slate-950 p-6 text-white">
            <div class="flex items-center gap-3">
              <span class="h-3 w-3 rounded-full bg-red-400"></span>
              <span class="h-3 w-3 rounded-full bg-amber-300"></span>
              <span class="h-3 w-3 rounded-full bg-emerald-400"></span>
            </div>
            <div class="mt-8 space-y-4">
              <div class="rounded-3xl bg-white/10 p-4">
                <div class="text-sm font-bold text-cyan-100">Todos</div>
                <div class="mt-2 h-3 w-3/4 rounded-full bg-white/25"></div>
                <div class="mt-2 h-3 w-1/2 rounded-full bg-white/15"></div>
              </div>
              <div class="rounded-3xl bg-white/10 p-4">
                <div class="text-sm font-bold text-cyan-100">Bookmarks</div>
                <div class="mt-2 h-3 w-4/5 rounded-full bg-white/25"></div>
                <div class="mt-2 h-3 w-2/3 rounded-full bg-white/15"></div>
              </div>
              <div class="rounded-3xl bg-white/10 p-4">
                <div class="text-sm font-bold text-cyan-100">Wallet</div>
                <div class="mt-2 h-3 w-2/3 rounded-full bg-white/25"></div>
                <div class="mt-2 h-3 w-1/2 rounded-full bg-white/15"></div>
              </div>
            </div>
            <div class="mt-8 rounded-3xl bg-white px-5 py-4 text-sm font-black text-slate-950">
              Fast. Responsive. Personal.
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script>
export default {
  name: 'AboutUs'
}
</script>
