# Runtime screenshots

Screenshots must be captured from the live Vite + readerd application with Chromium; no mock HTML or manually generated images are accepted. Save PNGs in this directory.

| Filename | View / theme | Resolution | Capture command |
|---|---|---:|---|
| `library-1280x720.png` | Library / Light | 1280×720 | `chromium --headless=new --disable-gpu --no-sandbox --hide-scrollbars --window-size=1280,720 --screenshot=apps/pocket-reader/docs/screenshots/library-1280x720.png http://127.0.0.1:5173` |
| `details-1280x720.png` | Book details / Light | 1280×720 | Navigate in the running app to a book before capture |
| `reader-epub-1280x720.png` | EPUB reader / Light | 1280×720 | Open a generated EPUB fixture through the library before capture |
| `reader-text-1280x720.png` | TXT reader / Light | 1280×720 | Open a generated TXT fixture through the library before capture |
| `bookmarks-1280x720.png` | Bookmarks / Light | 1280×720 | Navigate to Bookmarks before capture |
| `notes-1280x720.png` | Notes / Light | 1280×720 | Navigate to Notes before capture |
| `search-1280x720.png` | Search / Light | 1280×720 | Navigate to Search and enter a query before capture |
| `settings-1280x720.png` | Settings / Light | 1280×720 | Navigate to Settings before capture |
| `dark-1280x720.png` | Library / Dark | 1280×720 | Select Dark theme before capture |
| `eink-1280x720.png` | Library / E-Ink | 1280×720 | Select E-Ink theme before capture |
| `library-320x240.png` | Library / Light | 320×240 | Set Chromium window size to 320×240 |
| `library-480x320.png` | Library / Light | 480×320 | Set Chromium window size to 480×320 |
| `library-720x720.png` | Library / Light | 720×720 | Set Chromium window size to 720×720 |
| `library-1280x720-layout.png` | Library / Light | 1280×720 | Set Chromium window size to 1280×720 |

All names above specify the required capture set; entries remain pending until the genuine runtime PNG exists and its PNG IHDR width and height have been checked. DPI 1x, 1.5x, 2x and 3x checks also require a running application and should be recorded alongside the capture command when available.
