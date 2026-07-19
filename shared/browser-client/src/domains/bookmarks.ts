import type { BookmarkResponse, CreateBookmarkInput, UpdateBookmarkInput } from '@browser-server/shared-types'
import { type TokenProvider, apiFetch, buildQuery } from '../internals'

export function createBookmarkMethods(baseUrl: string, getToken?: TokenProvider) {
  return {
    getBookmarks(userId?: number, tags?: string, folderPath?: string): Promise<BookmarkResponse[]> {
      return apiFetch<BookmarkResponse[]>(
        baseUrl,
        'GET',
        `/api/bookmarks${buildQuery({ user_id: userId, tags, folder_path: folderPath })}`,
        undefined,
        getToken,
      )
    },

    createBookmark(data: CreateBookmarkInput): Promise<BookmarkResponse> {
      return apiFetch<BookmarkResponse>(baseUrl, 'POST', '/api/bookmarks', data, getToken)
    },

    updateBookmark(id: number, data: UpdateBookmarkInput): Promise<BookmarkResponse> {
      return apiFetch<BookmarkResponse>(baseUrl, 'PUT', `/api/bookmarks/${id}`, data, getToken)
    },

    deleteBookmark(id: number): Promise<void> {
      return apiFetch<void>(baseUrl, 'DELETE', `/api/bookmarks/${id}`, undefined, getToken)
    },
  }
}
