<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue';

type TokenKind = 'plain' | 'key' | 'string' | 'number' | 'literal' | 'punctuation' | 'invalid';
interface HighlightToken {
  kind: TokenKind;
  text: string;
}

const props = withDefaults(
  defineProps<{
    ariaLabel?: string;
    disabled?: boolean;
    fontSize?: number;
  }>(),
  {
    ariaLabel: 'JSON code editor',
    disabled: false,
    fontSize: 13,
  },
);

const emit = defineEmits<{
  save: [];
}>();

const content = defineModel<string>({ required: true });
const textarea = ref<HTMLTextAreaElement | null>(null);
const highlight = ref<HTMLElement | null>(null);
const scrollTop = ref(0);

const editorStyle = computed(() => ({
  '--editor-font-size': `${props.fontSize}px`,
  '--editor-line-height': `${Math.round(props.fontSize * 1.7)}px`,
}));
const tokens = computed(() => tokenizeJSON(content.value));
const lineNumbers = computed(() =>
  Array.from({ length: Math.max(1, content.value.split('\n').length) }, (_, index) => index + 1),
);

function tokenClass(kind: TokenKind): string {
  return `json-token-${kind}`;
}

function tokenizeJSON(source: string): HighlightToken[] {
  const result: HighlightToken[] = [];
  let index = 0;

  const push = (kind: TokenKind, start: number, end: number) => {
    result.push({ kind, text: source.slice(start, end) });
  };

  while (index < source.length) {
    const start = index;
    const char = source[index];

    if (/\s/.test(char)) {
      index += 1;
      while (index < source.length && /\s/.test(source[index])) index += 1;
      push('plain', start, index);
      continue;
    }

    if (char === '"') {
      index += 1;
      let terminated = false;
      while (index < source.length) {
        if (source[index] === '\\') {
          index = Math.min(source.length, index + 2);
          continue;
        }
        if (source[index] === '"') {
          index += 1;
          terminated = true;
          break;
        }
        index += 1;
      }
      let lookahead = index;
      while (lookahead < source.length && /\s/.test(source[lookahead])) lookahead += 1;
      push(terminated ? (source[lookahead] === ':' ? 'key' : 'string') : 'invalid', start, index);
      continue;
    }

    const number = source.slice(index).match(/^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/);
    if (number) {
      index += number[0].length;
      push('number', start, index);
      continue;
    }

    const literal = source.slice(index).match(/^(?:true|false|null)\b/);
    if (literal) {
      index += literal[0].length;
      push('literal', start, index);
      continue;
    }

    if ('{}[],:'.includes(char)) {
      index += 1;
      push('punctuation', start, index);
      continue;
    }

    index += 1;
    while (index < source.length && !/[\s"{}\[\],:]/.test(source[index])) index += 1;
    push('invalid', start, index);
  }

  return result;
}

function syncScroll() {
  const input = textarea.value;
  const layer = highlight.value;
  if (!input || !layer) return;
  scrollTop.value = input.scrollTop;
  layer.scrollTop = input.scrollTop;
  layer.scrollLeft = input.scrollLeft;
}

function replaceSelection(replacement: string, selectionStart: number, selectionEnd: number) {
  content.value = `${content.value.slice(0, selectionStart)}${replacement}${content.value.slice(selectionEnd)}`;
  void nextTick(() => {
    const input = textarea.value;
    if (!input) return;
    const caret = selectionStart + replacement.length;
    input.setSelectionRange(caret, caret);
    input.focus();
    syncScroll();
  });
}

function handleKeydown(event: KeyboardEvent) {
  const input = textarea.value;
  if (!input || props.disabled) return;

  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
    event.preventDefault();
    emit('save');
    return;
  }

  if (event.key !== 'Tab') return;
  event.preventDefault();
  const start = input.selectionStart;
  const end = input.selectionEnd;

  if (start === end) {
    replaceSelection('  ', start, end);
    return;
  }

  const lineStart = content.value.lastIndexOf('\n', start - 1) + 1;
  const block = content.value.slice(lineStart, end);
  if (event.shiftKey) {
    const unindented = block.replace(/^ {1,2}/gm, '');
    content.value = `${content.value.slice(0, lineStart)}${unindented}${content.value.slice(end)}`;
    void nextTick(() => {
      input.setSelectionRange(lineStart, lineStart + unindented.length);
      input.focus();
      syncScroll();
    });
    return;
  }

  const indented = block.replace(/^/gm, '  ');
  content.value = `${content.value.slice(0, lineStart)}${indented}${content.value.slice(end)}`;
  void nextTick(() => {
    input.setSelectionRange(lineStart, lineStart + indented.length);
    input.focus();
    syncScroll();
  });
}

watch(content, () => void nextTick(syncScroll));
onMounted(syncScroll);
</script>

<template>
  <div
    class="json-editor relative min-h-[470px] overflow-hidden bg-slate-950 transition focus-within:ring-1 focus-within:ring-violet-500/70 focus-within:ring-inset"
    :style="editorStyle"
  >
    <div
      aria-hidden="true"
      class="json-gutter absolute inset-y-0 left-0 z-20 w-12 overflow-hidden border-r border-white/10 bg-slate-950/95 text-slate-600 select-none"
    >
      <div class="pt-4 text-right" :style="{ transform: `translateY(-${scrollTop}px)` }">
        <span v-for="line in lineNumbers" :key="line" class="json-line-number block pr-3">{{
          line
        }}</span>
      </div>
    </div>

    <div class="absolute inset-y-0 right-0 left-12">
      <pre
        ref="highlight"
        aria-hidden="true"
        class="json-highlight pointer-events-none absolute inset-0 m-0 overflow-hidden p-4 whitespace-pre"
      ><code><span v-for="(token, index) in tokens" :key="index" :class="tokenClass(token.kind)">{{ token.text }}</span><span>&#8203;</span></code></pre>
      <textarea
        ref="textarea"
        v-model="content"
        :aria-label="ariaLabel"
        :disabled="disabled"
        wrap="off"
        spellcheck="false"
        autocomplete="off"
        class="json-input absolute inset-0 h-full w-full resize-none overflow-auto rounded-none border-0 bg-transparent p-4 whitespace-pre outline-none disabled:cursor-not-allowed disabled:opacity-60"
        @scroll="syncScroll"
        @keydown="handleKeydown"
      ></textarea>
    </div>
  </div>
</template>

<style scoped>
.json-editor {
  font-family: var(
    --font-code,
    'Fira Code',
    'JetBrains Mono',
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Consolas,
    monospace
  );
  font-variant-ligatures: contextual;
}

.json-highlight,
.json-input {
  font-size: var(--editor-font-size);
  line-height: var(--editor-line-height);
}

.json-gutter {
  font-size: max(10px, calc(var(--editor-font-size) - 2px));
  line-height: var(--editor-line-height);
}

.json-line-number {
  height: var(--editor-line-height);
  line-height: var(--editor-line-height);
}

.json-highlight {
  color: #cbd5e1;
  tab-size: 2;
}

.json-input {
  color: transparent;
  -webkit-text-fill-color: transparent;
  caret-color: #f8fafc;
  tab-size: 2;
  scrollbar-width: thin;
  scrollbar-color: rgb(100 116 139 / 55%) transparent;
}

.json-input::selection {
  background: rgb(124 58 237 / 48%);
  color: transparent;
  -webkit-text-fill-color: transparent;
}

.json-input::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}

.json-input::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 9999px;
  background: rgb(100 116 139 / 55%);
  background-clip: padding-box;
}

.json-token-key {
  color: #67e8f9;
}

.json-token-string {
  color: #a7f3d0;
}

.json-token-number {
  color: #fcd34d;
}

.json-token-literal {
  color: #c4b5fd;
  font-weight: 600;
}

.json-token-punctuation {
  color: #94a3b8;
}

.json-token-invalid {
  color: #fda4af;
  text-decoration: underline wavy #fb7185;
  text-underline-offset: 3px;
}
</style>
