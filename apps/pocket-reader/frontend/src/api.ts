export type Book = { id: number; title: string; author?: string; authors?: string[]; description?: string; language?: string; format?: string; cover_url?: string; cover?: string; tags?: string[]; favorite?: boolean; progress?: number; page_count?: number; file_size?: number; series?: string; publisher?: string; added_at?: string };
export type Progress = { book_id: number; chapter?: string | number; loc?: string | number; percent: number; updated_at?: string; device?: string };
export type Bookmark = { id: number; book_id: number; loc: string | number; chapter?: string; text?: string; created_at?: string };
export type Highlight = { id: number; book_id: number; loc: string | number; text: string; title?: string; color?: string; note?: string; created_at?: string };
export type Note = { id: number; book_id: number; loc?: string | number; text: string; title?: string; book_title?: string; created_at?: string };
export type Content = { book_id: number; loc: string | number; title?: string; content: string; content_type?: string; has_next: boolean; has_previous: boolean; next_loc?: string | number; previous_loc?: string | number };
export type Collection = { id: number; name: string; description?: string; book_ids?: number[]; book_count?: number };
export type Settings = { theme?: string; language?: string; font_family?: string; font_size?: number; line_height?: number; margin?: number; alignment?: string; page_turn?: string; [key: string]: unknown };
const base = '/api';
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${base}${path}`, { ...init, headers: { 'Content-Type': 'application/json', ...init?.headers } });
  if (!response.ok) { let message = `Request failed (${response.status})`; try { const body = await response.json(); message = body.error?.message || message; } catch { /* retain HTTP message */ } throw new Error(message); }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}
const json = (method: string, body?: unknown): RequestInit => ({ method, body: JSON.stringify(body) });
export const api = {
  books: async (q = '') => request<{items: Book[]; total: number; page: number; limit: number}>(`/books${q ? `?q=${encodeURIComponent(q)}` : ''}`),
  book: (id: number) => request<Book>(`/books/${id}`),
  home: () => request<Record<string, unknown>>('/home'),
  content: (id: number, loc?: string | number) => request<Content>(`/books/${id}/content${loc === undefined ? '' : `?loc=${encodeURIComponent(String(loc))}`}`),
  progress: (id: number) => request<Progress>(`/books/${id}/progress`),
  saveProgress: (id: number, data: Partial<Progress>) => request<Progress>(`/books/${id}/progress`, json('PUT', data)),
  favorite: (id: number, on: boolean) => request<void>(`/books/${id}/favorite`, { method: on ? 'POST' : 'DELETE' }),
  bookmarks: (bookId?: number) => request<{items: Bookmark[]}>(`/bookmarks${bookId ? `?book_id=${bookId}` : ''}`),
  addBookmark: (data: Partial<Bookmark>) => request<Bookmark>('/bookmarks', json('POST', data)),
  deleteBookmark: (id: number) => request<void>(`/bookmarks/${id}`, { method: 'DELETE' }),
  highlights: (bookId?: number) => request<{items: Highlight[]}>(`/highlights${bookId ? `?book_id=${bookId}` : ''}`),
  addHighlight: (data: Partial<Highlight>) => request<Highlight>('/highlights', json('POST', data)),
  notes: (bookId?: number, q = '') => request<{items: Note[]}>(`/notes${[bookId ? `book_id=${bookId}` : '', q ? `q=${encodeURIComponent(q)}` : ''].filter(Boolean).join('&') ? `?${[bookId ? `book_id=${bookId}` : '', q ? `q=${encodeURIComponent(q)}` : ''].filter(Boolean).join('&')}` : ''}`),
  addNote: (data: Partial<Note>) => request<Note>('/notes', json('POST', data)),
  updateNote: (id: number, data: Partial<Note>) => request<Note>(`/notes/${id}`, json('PUT', data)),
  deleteNote: (id: number) => request<void>(`/notes/${id}`, { method: 'DELETE' }),
  search: (q: string) => request<{items: Array<{book_id: number; book_title?: string; title?: string; text?: string; loc?: string | number}>}>(`/search?q=${encodeURIComponent(q)}`),
  collections: () => request<{items: Collection[]}>('/collections'),
  settings: () => request<Settings>('/settings'),
  saveSettings: (data: Settings) => request<Settings>('/settings', json('PUT', data)),
  notesExport: (format: 'md' | 'json') => request<unknown>(`/notes/export?format=${format}`),
  toc: (id: number) => request<{items: Array<{loc: string; title: string}>}>(`/books/${id}/toc`),
  chapterSearch: (id: number, q: string) => request<{items: Array<{loc: string; text: string; excerpt: string}>}>(`/books/${id}/search?q=${encodeURIComponent(q)}`),
  deleteHighlight: (id: number) => request<void>(`/highlights/${id}`, { method: 'DELETE' }),
};
