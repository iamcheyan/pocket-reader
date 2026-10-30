# Runtime screenshots

All files were captured from the live app (readerd :8791 + Vite :5199) with Chromium headless, using generated fixtures in `/tmp/pr-books` and `READERD_DATA=/tmp/pr-data2`. No placeholders or hand-written HTML were used. Dimensions were verified by reading each PNG's IHDR header.

Reproduce from the repository root (backend and frontend running):

```sh
cd apps/pocket-reader/docs/screenshots
chromium --headless=new --disable-gpu --no-sandbox --hide-scrollbars \
  --virtual-time-budget=9000 --window-size=1280,720 \
  --screenshot=library-1280x720.png \
  'http://127.0.0.1:5199/?theme=light&lang=en'
```

| File | View | Theme | Size (PNG IHDR) |
|---|---|---|---|
| `library-1280x720.png` | Library | Light | 1280×720 |
| `library-1920x1080.png` | Library | Light | 1920×1080 |
| `library-720x720.png` | Library (7-inch square) | Light | 720×720 |
| `library-480x320.png` | Library (6-inch landscape) | Light | 480×320 |
| `library-320x240.png` | Library (smallest) | Light | 320×240 |
| `details-1280x720.png` | Book details | Light | 1280×720 |
| `reader-epub-1280x720.png` | Reading view (EPUB) | Light | 1280×720 |
| `reader-pdf-1280x720.png` | Reading view (PDF) | Light | 1280×720 |
| `reader-320x240.png` | Reading view, compact | Light | 320×240 |
| `bookmarks-1280x720.png` | Bookmarks | Light | 1280×720 |
| `notes-1280x720.png` | Notes & highlights | Light | 1280×720 |
| `search-1280x720.png` | Search results | Light | 1280×720 |
| `settings-1280x720.png` | Settings incl. controls guide | Light | 1280×720 |
| `settings-eink-1280x720.png` | Settings | E-Ink | 1280×720 |
| `dark-1280x720.png` | Library | Dark | 1280×720 |
| `eink-1280x720.png` | Library | E-Ink | 1280×720 |
| `library-zh-1280x720.png` | Library | Light, 简体中文 | 1280×720 |
| `library-ja-1280x720.png` | Library | Light, 日本語 | 1280×720 |
| `dpi-1x-720x720.png` | Library, DPR 1 | Light | 720×720 |
| `dpi-1.5x-720x720.png` | Library, DPR 1.5 | Light | 1080×1080 |
| `dpi-2x-720x720.png` | Library, DPR 2 | Light | 1440×1440 |
| `dpi-3x-720x720.png` | Library, DPR 3 | Light | 2160×2160 |

## Dark vs E-Ink

`dark-1280x720.png` uses the dark palette (dark green-gray surfaces, soft shadows, rounded cards). `eink-1280x720.png` is grayscale, black-on-white, with square corners, no shadows or gradients, and no transitions. The two files differ in layout styling and are not interchangeable.

DPI screenshots use `--force-device-scale-factor=1|1.5|2|3` on a 720×720 CSS viewport; the resulting PNG pixel sizes are listed above.
