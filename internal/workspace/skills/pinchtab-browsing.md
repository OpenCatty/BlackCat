---
name: PinchTab Browsing
tags: [browsing, web, pinchtab, ulw]
---

# PinchTab Browsing

Saat user meminta browsing web atau riset online, gunakan tool `web`.

## Runtime behavior

- Jika `BLACKCAT_PINCHTAB_ENABLED=true` atau `BLACKCAT_PINCHTAB_BASE_URL` terisi, tool `web` akan memakai PinchTab API.
- Jika tidak aktif, tool `web` memakai HTTP fetch biasa.

## Env yang dipakai

- `BLACKCAT_PINCHTAB_ENABLED` (true/false)
- `BLACKCAT_PINCHTAB_BASE_URL` (default `http://127.0.0.1:9867` saat enabled)
- `BLACKCAT_PINCHTAB_TOKEN` (opsional, Bearer token)

## Rule penggunaan

1. Untuk web lookup biasa: panggil `web` dengan URL target.
2. Untuk mode `ulw`: tetap gunakan `web` sebagai jalur browsing utama.
3. Jangan mengeksekusi browser lokal langsung jika PinchTab sudah aktif.
