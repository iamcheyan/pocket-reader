// Reader client: library, progress and annotations over readerd.
// The device profile has no EPUB parser; content is fixture text for now.
import { createPortClient } from "./client-kit";
import type { AppConfig, Metric, Section, SettingRow } from "./model";

const sections: Section[] = [
  {
    id: "continue", title: "CONTINUE", jp: "続きを読む", items: [
      { id: "b1", title: "The Quiet Cartographer", subtitle: "Chapter 4 · The outer belt", meta: "62% · 12 min", badge: "EPUB", progress: 0.62, body: "The survey vessel held its course while the map redrew itself one star at a time.", lines: [{ label: "chapter", value: "4/12" }, { label: "format", value: "EPUB" }, { label: "pages", value: "218" }] },
      { id: "b2", title: "Notes on Light", subtitle: "Chapter 2 · Glass harbour", meta: "31% · 26 min", badge: "PDF", progress: 0.31, body: "A short essay on the colour of harbour water before sunrise.", lines: [{ label: "chapter", value: "2/8" }, { label: "format", value: "PDF" }, { label: "pages", value: "96" }] },
      { id: "b3", title: "Field Guide: Nebulae", subtitle: "Chapter 1 · Emission", meta: "8% · 41 min", badge: "TXT", progress: 0.08, body: "Emission nebulae are best observed with a narrow band filter.", lines: [{ label: "chapter", value: "1/14" }, { label: "format", value: "TXT" }, { label: "pages", value: "302" }] },
    ],
  },
  {
    id: "library", title: "LIBRARY", jp: "本棚", items: [
      { id: "c1", title: "Deep Field", subtitle: "12 books · 4 unread", meta: "collection", badge: "COL", body: "Long-form astronomy and exploration writing." },
      { id: "c2", title: "Bookmarks", subtitle: "7 saved passages", meta: "synced", badge: "MARK", body: "Bookmarks saved across the library." },
      { id: "c3", title: "Highlights", subtitle: "23 highlights · 4 notes", meta: "export md", badge: "NOTE", body: "Coloured highlights with optional notes." },
    ],
  },
];

const metrics: Metric[] = [
  { label: "BOOKS", value: "138" },
  { label: "UNREAD", value: "4" },
  { label: "MARKS", value: "7" },
];

const settings: SettingRow[] = [
  { id: "theme", label: "Theme", jp: "テーマ", value: "SEPIA", options: ["LIGHT", "DARK", "SEPIA", "EINK"] },
  { id: "size", label: "Text size", jp: "文字サイズ", value: "16", options: ["12", "14", "16", "18"] },
  { id: "flow", label: "Page turn", jp: "ページ送り", value: "SWIPE", options: ["SWIPE", "BUTTON"] },
];

export const APP: AppConfig = {
  title: "READER",
  jp: "読書",
  kind: "reader",
  page: "READER",
  tabs: ["本棚", "読書", "設定"],
  actions: ["PREV", "NEXT", "MARK", "NOTE"],
  settingsTitle: "READING",
  settingsJp: "読書設定",
  deviceGap: "readerd + EPUB missing.",
  sourceNote: "/api/books · /api/bookmarks",
};

export const client = createPortClient({
  resource: "reader",
  fixture: { sections, metrics, settings },
  endpoints: {
    list: "/api/books",
    get: (id) => `/api/books/${encodeURIComponent(id)}`,
    action: (name) => `/api/books/action/${encodeURIComponent(name.toLowerCase())}`,
  },
  act(name, payload, bundle) {
    const item = bundle.sections.flatMap((section) => section.items).find((entry) => entry.id === payload?.id);
    if (!item) return { ok: false, message: "no book selected" };
    if (name === "NEXT") { item.progress = Math.min(1, (item.progress ?? 0) + 0.06); return { ok: true, message: `${item.title}: page advanced` }; }
    if (name === "PREV") { item.progress = Math.max(0, (item.progress ?? 0) - 0.06); return { ok: true, message: `${item.title}: page back` }; }
    if (name === "MARK") { item.state = "bookmarked"; return { ok: true, message: `bookmark saved at ${Math.round((item.progress ?? 0) * 100)}%` }; }
    if (name === "NOTE") { item.state = "note"; return { ok: true, message: "note editor is a product TODO" }; }
    return { ok: false, message: `${name} unavailable` };
  },
});
