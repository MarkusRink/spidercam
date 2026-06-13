# Spidercam web UIs

SolidJS frontends for the host console and participant monitor. During development they run as Vite dev servers that proxy API traffic to the mock server (or `spidercamd` in production).

## Prerequisites

From the repository root:

```bash
npm install
```

## Dev stack (mock server + both UIs)

```bash
npm run dev
```

This starts three processes:

| Process     | URL                    | API backend              |
| ----------- | ---------------------- | ------------------------ |
| Host UI     | http://127.0.0.1:5175/ | mock host `:1235`        |
| Participant | http://127.0.0.1:5174/ | mock participant `:1234` |
| Mock server | `:1235` + `:1234`      | —                        |

Vite proxies `/api` (and `/dev` on the host port only) to the matching mock listener.

## Two-browser UX test (manual)

1. Run `npm run dev` from the repo root.
2. Open **host**: http://127.0.0.1:5175/
3. Open **participant** in a second window or profile: http://127.0.0.1:5174/
4. On the participant page, set a display name and click **Connect**.
5. On the host page, confirm the stream count increases and mixer state updates.

Use **copy URL** on the host header to grab the participant link (`http://127.0.0.1:1234/` in mock mode).

Scenario helpers (`POST /dev/scenario/*`) are only served on the host mock port (`:1235`) and are proxied from the host Vite dev/preview server — not from the participant UI port.

## Automated UI tests

```bash
npm run test:ui
```

Playwright starts the mock server plus `vite preview` on `:4175` (host) and `:4174` (participant), with fake media device flags for Chromium.

## Build

```bash
npm run build
```

Builds all workspaces (`web/protocol`, `web/ui-theme`, both apps, mock server).
