import type { AIMessage, ChatQuestion } from '@browser-server/shared-types';

/**
 * Tool-message envelope parsing + presentation derivation (labels, status,
 * sections). Tool messages store `{ tool, args, result, decision }` as a JSON
 * string in `content`.
 */

export interface ToolData {
  name: string;
  args: unknown;
  result: unknown;
  decision: 'approved' | 'rejected' | 'commented' | 'answered' | null;
}

export interface ToolSection {
  label: string;
  content: string;
  copyValue: string;
}

export interface ToolStatusInfo {
  label: string;
  /** lucide icon key understood by the renderer: success | warn | danger | running */
  tone: 'success' | 'warn' | 'danger' | 'running';
  className: string;
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function formatJson(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (typeof value === 'string') return value;
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

/** Parse a tool message's JSON envelope (fault-tolerant). */
export function parseToolContent(msg: Pick<AIMessage, 'content'>): {
  tool?: string;
  args?: unknown;
  result?: unknown;
  decision?: string;
} {
  try {
    return JSON.parse(msg.content);
  } catch {
    return {};
  }
}

export function deriveToolData(message: AIMessage): ToolData {
  if (message.role !== 'tool') return { name: '', args: null, result: null, decision: null };
  try {
    const parsed = JSON.parse(message.content);
    return {
      name: parsed.tool || '',
      args: parsed.args ?? null,
      result: parsed.result ?? parsed,
      decision: parsed.decision ?? null,
    };
  } catch {
    return { name: '', args: null, result: message.content, decision: null };
  }
}

/** "execute_command" → "Shell"; snake_case → Title Case. */
export function toolLabel(name: string): string {
  if (name === 'execute_command') return 'Shell';
  if (name === 'retry_tool_call') return 'tool-call recovery';
  if (!name) return 'Tool';
  return name
    .split('_')
    .filter(Boolean)
    .map((word) => word[0].toUpperCase() + word.slice(1))
    .join(' ');
}

export function toolStatus(message: AIMessage, data: ToolData): ToolStatusInfo {
  const isRetry = data.name === 'retry_tool_call';
  if (message.status === 'pending' && !data.decision) {
    return {
      label: isRetry ? 'resume required' : 'approval required',
      tone: 'warn',
      className: 'text-amber-600 dark:text-amber-400',
    };
  }
  if (data.decision === 'commented') {
    return {
      label: 'commented',
      tone: 'warn',
      className: 'text-amber-600 dark:text-amber-400',
    };
  }
  if (data.decision === 'answered') {
    return {
      label: 'answered',
      tone: 'success',
      className: 'text-emerald-600 dark:text-emerald-400',
    };
  }
  if (message.status === 'pending') {
    return {
      label: isRetry ? 'resuming' : 'running',
      tone: 'running',
      className: 'text-blue-600 dark:text-blue-400',
    };
  }
  const record = isRecord(data.result) ? data.result : null;
  const exitCode = record?.exit_code;
  const failed =
    message.status === 'error' ||
    Boolean(record?.error) ||
    (typeof exitCode === 'number' && exitCode !== 0);
  if (failed) {
    const rejected = record?.error === 'rejected by user';
    return {
      label: rejected ? 'rejected' : 'failed',
      tone: 'danger',
      className: 'text-red-600 dark:text-red-400',
    };
  }
  return { label: '', tone: 'success', className: 'text-emerald-600 dark:text-emerald-400' };
}

const isFinished = (message: AIMessage, data: ToolData) =>
  message.status !== 'pending' || data.decision === 'commented';

/** Command/stdout/stderr for execute_command; Arguments/Result otherwise. */
export function toolSections(message: AIMessage, data: ToolData): ToolSection[] {
  const record = isRecord(data.result) ? data.result : null;

  if (data.name === 'execute_command' && isRecord(data.args)) {
    const command = typeof data.args.command === 'string' ? data.args.command : '';
    const stdout = record?.stdout;
    const stderr = record?.stderr;
    const error = record?.error;
    const stderrText = [
      typeof stderr === 'string' ? stderr.trimEnd() : '',
      typeof error === 'string' ? error : '',
    ]
      .filter(Boolean)
      .join('\n');

    const sections: ToolSection[] = [];
    if (command) sections.push({ label: 'Command', content: `$ ${command}`, copyValue: command });
    if (isFinished(message, data) && typeof stdout === 'string') {
      sections.push({ label: 'Stdout', content: stdout || '(no output)', copyValue: stdout });
    }
    if (isFinished(message, data) && (typeof stderr === 'string' || typeof error === 'string')) {
      sections.push({
        label: 'Stderr',
        content: stderrText || '(no output)',
        copyValue: stderrText,
      });
    }
    return sections;
  }

  const sections: ToolSection[] = [];
  if (data.args !== null && data.args !== undefined) {
    const args = formatJson(data.args);
    sections.push({ label: 'Arguments', content: args, copyValue: args });
  }
  if (isFinished(message, data) && data.result !== null && data.result !== undefined) {
    const result = formatJson(data.result);
    sections.push({ label: 'Result', content: result || '(no output)', copyValue: result });
  }
  return sections;
}

/* ------------------------------ ask_questions ------------------------------ */

export function isChatQuestion(value: unknown): value is ChatQuestion {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.prompt !== 'string')
    return false;
  return (
    value.kind === undefined ||
    ['text', 'choice', 'multi_choice', 'multiple_choice', 'confirm'].includes(value.kind as string)
  );
}

export interface QuestionRequest {
  context: string;
  questions: ChatQuestion[];
}

export function parseQuestionRequest(data: ToolData): QuestionRequest | null {
  if (data.name !== 'ask_questions' || !isRecord(data.args)) return null;
  const questions = Array.isArray(data.args.questions) ? data.args.questions : [];
  return {
    context: typeof data.args.context === 'string' ? data.args.context : '',
    questions: questions.filter(isChatQuestion).map((question) => ({
      id: question.id,
      prompt: question.prompt,
      kind: question.kind,
      options: question.options,
      default: question.default,
      required: question.required,
    })),
  };
}

/** The retry-tool-call message carried in args.message. */
export function retryMessage(data: ToolData): string {
  if (data.name !== 'retry_tool_call' || !isRecord(data.args)) return '';
  return typeof data.args.message === 'string' ? data.args.message : '';
}

/* ------------------------- tools-panel history entries ------------------------- */

export interface ToolCallEntry {
  id: string;
  name: string;
  status: string;
  args?: string;
  result?: string;
}

/** Derive the tools-panel "History" entries from the message list. */
export function deriveToolCallEntries(messages: AIMessage[]): ToolCallEntry[] {
  return messages
    .filter((m) => m.role === 'tool')
    .map((m) => {
      let name = 'Tool call';
      let args: string | undefined;
      let result: string | undefined;
      let status = m.status === 'pending' ? 'pending' : 'completed';
      try {
        const parsed = JSON.parse(m.content);
        name = parsed.tool || name;
        if (parsed.args) args = formatJson(parsed.args);
        if (parsed.result !== null && parsed.result !== undefined) {
          result = formatJson(parsed.result);
        }
        if (parsed.decision === 'rejected') status = 'rejected';
        else if (parsed.decision === 'commented') status = 'commented';
        else if (isRecord(parsed.result) && parsed.result.error) status = 'error';
        else if (m.status === 'completed') status = 'completed';
        else if (m.status === 'error') status = 'error';
      } catch {
        /* fall through with defaults */
      }
      return { id: m.tool_call_id || m.id, name, status, args, result };
    });
}

/** msg-scoped wrappers used by the memory explorer cards. */
export function getToolArgs(msg: Pick<AIMessage, 'content'>): string {
  const parsed = parseToolContent(msg);
  if (parsed.args === null || parsed.args === undefined) return '';
  return formatJson(parsed.args);
}

export function getToolResult(msg: Pick<AIMessage, 'content'>): string {
  const parsed = parseToolContent(msg);
  if (parsed.result === null || parsed.result === undefined) return '';
  return formatJson(parsed.result);
}

export function getToolDecision(msg: Pick<AIMessage, 'content'>): string {
  return parseToolContent(msg).decision || '';
}
