# DEVICE VERIFICATION — pocket-reader

Status: **DEVICE-PASS** for the captured states. Local host verification is in
`apps/pocket-reader/POCKETJS-VERIFICATION.md`.

## Device

- Anbernic RG34XX, Base OS 1.5.2 (h700), kernel 4.9.170 aarch64.
- Access: Debian host -> HX90 -> USB ADB.
- Display: `/dev/fb0`, 720x480 visible page, 32 bpp.
- Runtime: `/mnt/sdcard/System/pocketjs/bin/pocket-linux-kms` (existing boot runtime, not rebuilt or replaced).
- Package: `pocket-reader` sha256 `a06d0ea961d82e4fdce077aafce7f9e04bd61f6c5cbf55e2114b63ee8de855f9`.
- Deployed to `/storage/pjs/pocket-reader/app.pocket` only.

## Procedure

1. Recorded boot launcher pid `28102` and `boot.pocket`
   sha256 `6ed41612bce1723d1eda64b5a0ee9333a105ed498b7085c03267539cf6c1b32d`.
2. Paused the launcher supervisor, stopped the launcher, and started the
   package with the same runtime and `/dev/fb0`.
3. Created a virtual keyboard with `/dev/uinput` and drove the native view
   tree through a FIFO (`uinputd-c`); no host browser and no HTML.
4. Captured `/dev/fb0` with `dd` -> base64 -> PIL.
5. Killed the package, restored the launcher, and read the pid
   `21225` and boot hash back.

Result: launcher pid changed, boot package sha256 stayed
`6ed41612bce1723d1eda64b5a0ee9333a105ed498b7085c03267539cf6c1b32d`. No write to `/storage/System`, `/storage/pocketjs`,
`/storage/baseos.conf` or `/storage/content`.

## Frames

The images are the device framebuffer. The UI was rasterized by the PocketJS
native renderer on the RG34XX, not by host Chrome.

| file | sha256 (first 16) |
|---|---|
| `docs/device/device-reader-detail-action.png` | `f0f7cd14886c8c3b` |
| `docs/device/device-reader-detail.png` | `435902c5f6986d6f` |
| `docs/device/device-reader-fb-after.png` | `d59130e3329b8496` |
| `docs/device/device-reader-fb-before.png` | `d59130e3329b8496` |
| `docs/device/device-reader-home-focus.png` | `24a3242b8456d81f` |
| `docs/device/device-reader-home.png` | `3f4d0296947c27f1` |
| `docs/device/device-reader-settings-toggled.png` | `c4618e3f032d4bd6` |
| `docs/device/device-reader-settings.png` | `487bfb478683f6af` |

`device-reader-fb-before.png` and `device-reader-fb-after.png` are the boot
launcher before and after; they are byte-identical here, which is the intended
restore evidence.

Captured states: 6 states: home, focus, detail, detail action, settings, settings toggle.
