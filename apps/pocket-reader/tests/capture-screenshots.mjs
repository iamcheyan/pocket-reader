// Capture the Pocket Reader screenshot matrix from the live app.
//
//   READERD_URL=http://127.0.0.1:8791 \
//   PUPPETEER_CORE=$HOME/.bun/install/global/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js \
//   CHROME_BIN=/usr/bin/google-chrome node tests/capture-screenshots.mjs
//
// State is reached through the app's deterministic deep-link parameters
// (?screen=…&book=…&theme=…&lang=…&q=…), each capture is a fresh navigation.
// DPI 1x is intentionally omitted: it is byte-identical to library-720x720.
// Fails on duplicate hash, wrong PNG size, horizontal overflow, missing text,
// or any console/HTTP error. Writes docs/screenshots/manifest.json.
import { isAbsolute } from 'node:path';
import { createHash } from 'node:crypto';
import { mkdirSync, readFileSync } from 'node:fs';
import { writeFile } from 'node:fs/promises';
import path from 'node:path';

const puppeteerPath = process.env.PUPPETEER_CORE;
const puppeteer = (await import(
  puppeteerPath && isAbsolute(puppeteerPath) ? puppeteerPath : 'puppeteer-core'
)).default;

const BASE = (process.env.READER_UI ?? 'http://127.0.0.1:5199').replace(/\/$/, '');
const CHROME = process.env.CHROME_BIN ?? '/usr/bin/google-chrome';
const OUT = path.resolve(process.env.READER_SHOTS ?? 'apps/pocket-reader/docs/screenshots');
const EPUB_ID = process.env.READER_EPUB_ID ?? '1';
const PDF_ID = process.env.READER_PDF_ID ?? '2';
const sleep = ms => new Promise(r => setTimeout(r, ms));

mkdirSync(OUT, { recursive: true });

// name, css width, css height, dsf, query string
const SHOTS = [
  ['library-1280x720', 1280, 720, 1, 'screen=library&theme=light&lang=en'],
  ['library-1920x1080', 1920, 1080, 1, 'screen=library&theme=light&lang=en'],
  ['library-720x720', 720, 720, 1, 'screen=library&theme=light&lang=en'],
  ['library-480x320', 480, 320, 1, 'screen=library&theme=light&lang=en'],
  ['library-320x240', 320, 240, 1, 'screen=library&theme=light&lang=en'],
  ['details-1280x720', 1280, 720, 1, `book=${EPUB_ID}&theme=light&lang=en`],
  ['reader-epub-1280x720', 1280, 720, 1, `screen=reader&book=${EPUB_ID}&theme=light&lang=en`],
  ['reader-pdf-1280x720', 1280, 720, 1, `screen=reader&book=${PDF_ID}&theme=light&lang=en`],
  ['reader-320x240', 320, 240, 1, `screen=reader&book=${EPUB_ID}&theme=light&lang=en`],
  ['bookmarks-1280x720', 1280, 720, 1, 'screen=bookmarks&theme=light&lang=en'],
  ['notes-1280x720', 1280, 720, 1, 'screen=notes&theme=light&lang=en'],
  ['search-1280x720', 1280, 720, 1, 'screen=search&q=Volume&theme=light&lang=en'],
  ['settings-1280x720', 1280, 720, 1, 'screen=settings&theme=light&lang=en'],
  ['settings-eink-1280x720', 1280, 720, 1, 'screen=settings&theme=eink&lang=en'],
  ['dark-1280x720', 1280, 720, 1, 'screen=library&theme=dark&lang=en'],
  ['eink-1280x720', 1280, 720, 1, 'screen=library&theme=eink&lang=en'],
  ['library-zh-1280x720', 1280, 720, 1, 'screen=library&theme=light&lang=zh'],
  ['library-ja-1280x720', 1280, 720, 1, 'screen=library&theme=light&lang=ja'],
  ['dpi-1.5x-720x720', 720, 720, 1.5, 'screen=library&theme=light&lang=en'],
  ['dpi-2x-720x720', 720, 720, 2, 'screen=library&theme=light&lang=en'],
  ['dpi-3x-720x720', 720, 720, 3, 'screen=library&theme=light&lang=en'],
];

const errors = [];
const browser = await puppeteer.launch({
  executablePath: CHROME,
  headless: 'new',
  args: ['--no-sandbox', '--disable-gpu', '--hide-scrollbars', '--disable-dev-shm-usage'],
});
const page = await browser.newPage();
page.on('console', (message) => { if (message.type() === 'error') errors.push(`console: ${message.text()}`); });
page.on('pageerror', (error) => errors.push(`pageerror: ${error.message}`));
page.on('response', (response) => { if (response.status() >= 400) errors.push(`http ${response.status()}: ${response.url()}`); });

const seen = new Map();
const manifest = [];

try {
  for (const [name, width, height, dsf, query] of SHOTS) {
    await page.setViewport({ width, height, deviceScaleFactor: dsf });
    await page.goto(`${BASE}/?${query}`, { waitUntil: 'networkidle2', timeout: 30000 });
    await sleep(name.startsWith('reader') ? 1400 : 800);
    const audit = await page.evaluate(() => ({
      overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
      text: document.body.innerText.trim().length,
    }));
    if (audit.overflow > 1) throw new Error(`${name}: horizontal overflow ${audit.overflow}px`);
    if (audit.text < 60) throw new Error(`${name}: page rendered almost no text (${audit.text})`);
    const file = path.join(OUT, `${name}.png`);
    await page.screenshot({ path: file });
    const bytes = readFileSync(file);
    if (bytes.subarray(0, 8).toString('binary') !== '\x89PNG\r\n\x1a\n') throw new Error(`${name}: not a PNG`);
    const w = bytes.readUInt32BE(16);
    const h = bytes.readUInt32BE(20);
    const expectedW = Math.round(width * dsf);
    const expectedH = Math.round(height * dsf);
    if (w !== expectedW || h !== expectedH) throw new Error(`${name}: PNG ${w}x${h}, expected ${expectedW}x${expectedH}`);
    const sha = createHash('sha256').update(bytes).digest('hex');
    if (seen.has(sha)) throw new Error(`${name}: duplicate of ${seen.get(sha)}`);
    seen.set(sha, name);
    manifest.push({ name: `${name}.png`, width: w, height: h, css: `${width}x${height}`, dsf, query, sha256: sha });
    console.log(`${name}: ${w}x${h} sha256=${sha.slice(0, 16)}`);
  }
} finally {
  await page.close();
  await browser.close();
}

if (errors.length) {
  throw new Error(`console/HTTP errors during capture:\n${[...new Set(errors)].join('\n')}`);
}
await writeFile(path.join(OUT, 'manifest.json'), `${JSON.stringify({ captured_at: new Date().toISOString(), ui: BASE, chrome: CHROME, screenshots: manifest }, null, 2)}\n`);
console.log(`captured ${manifest.length} screenshots, ${seen.size} unique hashes, no overflow, no console errors`);
