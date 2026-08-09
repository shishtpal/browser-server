// Barrel for the app-facing API layer. Domain modules live alongside this file;
// importing from `lib/api` continues to resolve here so existing callers are unchanged.
export { API_BASE } from './client';
export * from './health';
export * from './todos';
export * from './bookmarks';
export * from './history';
export * from './wallet';
export * from './users';
export * from './analytics';
export * from './ai';
export * from './memory';
export * from './prompts';
export * from './quiz';
