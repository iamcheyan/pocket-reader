import { readFileSync } from 'node:fs';

const source = readFileSync(new URL('../frontend/src/i18n.ts', import.meta.url), 'utf8');
const localeObjects = [...source.matchAll(/\b(en|zh|ja):\s*\{([^{}]*)\}/gs)];
if (localeObjects.length !== 3) throw new Error(`Expected 3 locale dictionaries, found ${localeObjects.length}`);
const dictionaries = new Map(localeObjects.map(([, locale, body]) => [locale, new Set([...body.matchAll(/(?:^|,)\s*([A-Za-z][\w]*):/g)].map(([, key]) => key))]));
const reference = dictionaries.get('en');
for (const locale of ['zh', 'ja']) {
  const keys = dictionaries.get(locale);
  const missing = [...reference].filter(key => !keys.has(key));
  const extra = [...keys].filter(key => !reference.has(key));
  if (missing.length || extra.length) throw new Error(`${locale} locale mismatch; missing=${missing} extra=${extra}`);
}
console.log(`Locale keys match across en/zh/ja (${reference.size} keys).`);
