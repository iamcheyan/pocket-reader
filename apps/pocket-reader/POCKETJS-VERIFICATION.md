# POCKETJS VERIFICATION — pocket-reader

Round: PocketJS refactor. Device acceptance is a later pass.

## Shape

- Pocket app: `apps/pocket-reader/pocket/`
- Frozen React reference: `apps/pocket-reader/frontend/` (design reference; not extended).
- TSX files in the port: **6** (`main.tsx`, `app.tsx`, `ui.tsx`, three screens).
- Native tree only: `View`/`Text` from `@pocketjs/framework/solid/components`; no DOM, iframe, texture screenshot, or `dangerouslySetInnerHTML`.

## Manifest

- id: `dev.pocket-reader.pocket`
- entry: `src/main.tsx`
- output: `pocket-reader`
- viewport: `360×240 @2x` → physical `720×480`
- capabilities: `text.glyphs.baked`, `input.buttons`
- target: `rocknix-h700-kms`

## Admission / compile

```sh
cd apps/pocket-reader/pocket
./setup-sdk.sh
SDK=${POCKETJS_SDK:-$HOME/development/.pocketjs-star-drift-sdk}
bun "$SDK/tools/pocket.ts" check \
  --runtime-config "$POCKET_VERIFY/pjs/profiles/pocket.runtime-config.ts" \
  --target rocknix-h700-kms --manifest pocket.json
bun "$SDK/tools/pocket.ts" compile \
  --runtime-config "$POCKET_VERIFY/pjs/profiles/pocket.runtime-config.ts" \
  --target rocknix-h700-kms --manifest pocket.json --output out.pocket
```

Admission: `sha256:b57e4fec53dfa19730131016fb60c2db1d74cfd843ba6587166a3a53717cdbcc`; TypeScript passes for the app module graph. Package: `out.pocket`, sha256 `780231c253b73aeda4b2f98402d5ef2dcedd01835a37310226054029033460d3`.

## Frames

Rendered with the SDK wasm core at `renderScaled(2)` (720×480), not a browser:

```sh
cd "$POCKET_VERIFY"
bun tools/render-pocket.ts \
  --dir repos/pocket-reader/apps/pocket-reader/pocket \
  --script repos/pocket-reader/apps/pocket-reader/pocket/frames.json \
  --out repos/pocket-reader/docs/pocket
```

| file | state | sha256 (first 16) |
|---|---|---|
| `docs/pocket/screen-home.png` | home | `97d2fbb65b5c60a3` |
| `docs/pocket/screen-home-focus.png` | home-focus | `4263eacaaced756b` |
| `docs/pocket/screen-detail.png` | detail | `52be7faa499f5d23` |
| `docs/pocket/screen-detail-action.png` | detail-action | `8e06d78e9b4ae2af` |
| `docs/pocket/screen-settings.png` | settings | `6e069425029fc1fa` |
| `docs/pocket/screen-settings-toggled.png` | settings-toggled | `7b8b0abcafb8246f` |

All six files have distinct bytes; the renderer rejects duplicate frames. The
tape is: move focus, open the item, run the first detail action, switch to
settings, move a row, toggle it.

Screens: Continue/library, reading view, reading settings. Interactions: PREV/NEXT move progress; MARK saves a bookmark; NOTE reports the editor TODO.

## Adapter

- **fixture** — this port runs from JSON baked into the package and labels the
  header `fixture`. `client.ts` also contains the `http` and `bridge` adapters
  behind the same `list()/get()/action()` surface.
- `http` uses `@pocketjs/framework/net-api`; valid only in the host dev profile.
- `bridge` uses `host.__pvc(JSON.stringify(request))`; the runtime forwards to
  the Go daemon on `127.0.0.1:<port>` with a hard timeout.
- Device gap: readerd bridge plus EPUB/PDF parsing. Annotations stay in memory.
- Daemon contract: /api/books, /api/books/:id/content, /api/bookmarks; demo catalog.

## Not done / open questions

- Device run, saves/annotations and the bridge itself are out of this round.
- Real media/audio, OAuth, emulator, engine and system writes stay unimplemented
  where the original product already marked them mock or unavailable.
- Long lists use the six-item fixture surface; paging/virtual lists are a
  follow-up once the bridge provides paginated data.
