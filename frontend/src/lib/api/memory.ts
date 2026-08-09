import type {
  AIMemoryStats,
  AIMemoryGraph,
  AIMemoryFragment,
  AIMemoryWriteOp,
  AIMemoryWriteResult,
} from '@browser-server/shared-types';
import { client } from './client';

export function getAIMemoryStats(): Promise<AIMemoryStats> {
  return client.getAIMemoryStats();
}

export function getAIMemoryGraph(): Promise<AIMemoryGraph> {
  return client.getAIMemoryGraph();
}

export function getAIMemoryFragment(id: string): Promise<AIMemoryFragment> {
  return client.getAIMemoryFragment(id);
}

export function writeAIMemory(ops: AIMemoryWriteOp[]): Promise<AIMemoryWriteResult> {
  return client.writeAIMemory(ops);
}

export function maintainAIMemory(): Promise<Record<string, unknown>> {
  return client.maintainAIMemory();
}
