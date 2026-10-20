# 🔒 PROJECT LOCK — ChitChat-App (only project)

**Locked on:** 2026-10-20 (Asia/Dhaka)
**Owner rule:** this repository works on **one project only — `ChitChat-App.zip`**.
Nothing else gets added, mixed in, or refactored "on the side".

---

## 1. What this project is

Windows desktop app **ChitChat Bot** — Go + Wails v2 (Go 1.25 / Wails 2.12.0),
with the **EVAChat Flow** message engine (TXT input/output rule packs).

```
chatpersona/                    <- repo root == project root
├── install.bat                 <- owner runs this on Windows (install + build)
├── run-dashboard.bat           <- owner runs this to start the built app
└── source_code/                <- the whole Go/Wails source
    ├── app.go, main.go, live_chat.go, home_chat.go, show_log.go
    ├── internal/               <- bot, browser, api, proxy, uistore, model, logwall
    ├── cmd/                    <- chitchatui (dashboard), chitchatbot (CLI worker)
    ├── apitest/                <- standalone OpenAI-compatible chat tester
    ├── capture/                <- manual traffic recorder (not app runtime)
    ├── docs/                   <- PROJECT_MAP, PROJECT_MEMORY, guides
    ├── scripts/                <- verify / package / bundle tooling
    ├── tests/                  <- node test suite
    ├── build/                  <- appicon.png + windows/ wails resources
    ├── install.bat, verify.bat, run-dashboard.bat
    └── go.mod, go.sum, wails.json, package.json
```

Read before editing: `source_code/AGENTS.md`, `source_code/docs/PROJECT_MAP.md`,
`source_code/docs/PROJECT_MEMORY.md`.

---

## 2. What the agent may do here

- edit / code / debug / fix errors **inside this project only**
- add or fix tests, docs, and TXT reply packs
- rebuild the clean delivery zip after every change
- **never** ship executables, runtime data, logs, caches or `node_modules`

## 3. What must NEVER enter the repo or the delivery zip

```
data/                     (profiles, webview cache, badger DB, eva_chat runtime copy)
build/bin/*.exe           (chitchatui.exe, chitchatbot.exe)
build/install.log
node_modules/
verification-output/
capture/capture-sessions/ capture/profiles/
logs.txt, *.log, *.tmp, *.zip (other than the delivery zip)
```
These are runtime/private state: accounts, cookies, sessions, browser profiles.
They stay on the owner's machine and are re-created by `install.bat` / first run.

## 4. Delivery contract — "clean zip"

Every time code changes, the agent produces **one** file:

```
dist/ChitChat-App-clean-YYYY-MM-DD_HHMM.zip
```

Contents = `install.bat`, `run-dashboard.bat`, `source_code/**` **only working
source files**. Target size: **≈1 MB** (never tens of MB).

Built by `_packaging/make-clean-zip.sh` (allowlist, not guesswork).
The owner then: extract → run `install.bat` → run `run-dashboard.bat`.

## 5. Out of scope for this repository

- any other project, repo, Drive file or dataset
- publishing/pushing to other repositories without an explicit request
- changing unrelated original behaviour (modes 1 & 2, license, proxy, DB schema)

---

## 6. Journal (append newest first)

**2026-10-20 — API Chat (new separate mode) + EVAChat Flow is pure TXT**

- Owner rule: **API Chat** and **EVAChat Flow** are two separate selectable
  options in Message; the AI block inside EVAChat Flow is gone.
- New `source_code/internal/apichat/` (director, signals, sanitizer, prompt,
  OpenAI-compatible client, session, engine) + `internal/bot/api_chat.go`
  glue + `api_assets/{persona,persona_countries,share,fallback,README}.txt`
  seeded to `data/api_chat/` without overwriting edits.
- Dashboard: 4th "API Chat" option on **both** Show Mode and Background Bot
  Message pages, with base URL / model / key / temperature / share-turn
  fields, Test API and Open Data Folder. `TestEVAChatAI` → `TestAPIChat`.
- Persistence: `AccountState.APIChatSessions` (history + director + persona).
  Snap username is appended by Go, never by the model; missing key refuses to
  start in API mode.
- Verified: `npm test` **74/74 PASS**, `node --check`, wasm gofmt,
  tree-sitter 93 Go files / 0 syntax errors. No Go toolchain in the sandbox —
  first compile is on the owner's Windows PC via `install.bat`.

**2026-10-20 — Background Bot → 5 · Live Logs (new feature)**

- Owner asked for the Show Mode "Live Logs" section in Background Bot.
- Added `source_code/bot_log.go` + `bot_log_test.go` (500-entry feed,
  `GetBotLiveLog` / `ClearBotLiveLog`), wired the worker `Logf`/`Activityf`
  callbacks and bot lifecycle/ban events into it.
- Dashboard: new `Live Logs` nav item + `bg-logs` subpanel
  (`bg-log-list`, `bg-log-filter`, `bg-log-clear`, `bg-log-empty`), same
  All/SMS/Rules/Errors filter, 1 s polling only while the tab is open.
- Tests: `npm test` **72/72 PASS** (2 new UI tests, 3 new Go tests),
  Go syntax parsed clean (82 files).

**2026-10-20 — project imported & locked**

- Source: Drive `ChitChat-App.zip` (19 Oct 2026, 60.2 MB, 2144 files) →
  extracted at repo root, exactly as the owner ships it (`install.bat` +
  `run-dashboard.bat` + `source_code/`).
- Stripped before committing: `source_code/data/` (71 MB: WebView2 cache,
  badger DB, profiles, seeded `eva_chat`), `build/bin/chitchatui.exe` (25 MB),
  `build/install.log`, `node_modules/` (17 MB), `logs.txt`, empty
  `verification-output/`.
  **114 MB / 2144 files → 1.45 MB / 215 files (delivery zip 584 KB).**
- Verified byte-for-byte against the original for all 213 kept source files —
  zero content differences.
- `npm ci --ignore-scripts` + `npm test` in `source_code/`: **69 / 69 pass**.
- No Go toolchain in the agent sandbox → real `go build` / `wails build` stays
  on the owner's Windows machine via `install.bat`.

---

_If a task needs something outside this boundary, the agent stops and asks first._

**2026-10-20 — API Chat quality phase**

- Phase 1–4 of `API_CHAT_QUALITY_PLAN.md` implemented inside the same locked
  project: understanding/ledger/repeat/guard/style/goal layers, prompt v2,
  sharesnap deck + rotation, snap-dodge close, tests and docs.
- EVAChat TXT engine, browser/API/proxy layers and every other mode are
  untouched; no new runtime dependency and no new UI control.

**2026-10-20 — local test chat (final task)**

- Added `test.bat` (root + `source_code/`) and the offline console harness
  (`cmd/chitchattest` + `internal/bot/localchat.go` + tests + docs). It covers
  the dashboard's four Message options, reads the saved settings, never sends
  to the real app and never writes bot state. Still the same locked project;
  no new runtime dependency for the dashboard and no UI change.

**2026-10-20 — API Chat snap hygiene (owner rule)**

- The bot never says snap/sc/snp/"add me" before the share; the only mention is
  the final `sharesnap.txt` line. A hand-written tease (hot/playful/soft) is sent
  right before it. Same locked project; no new dependency, no new UI fields.

**2026-10-20 — একটাই tease, তারপর share**

- API Chat-এর share টার্ন এখন পুরো হাতে-লেখা: `[tease, share]` — মাঝে কোনো
  মডেল কল নেই। `sharesnap.txt`-এ ১৮ লাইনের মধ্যে মাত্র ৪টায় snap/sc,
  তাই প্রতি চ্যাটে app-নাম সর্বোচ্চ একবার।
- পুরোনো ডেক শুধু untouched হলে একবার auto-upgrade; মালিকের এডিট safe।
- প্রজেক্ট লক অপরিবর্তিত: শুধু `ChitChat-App.zip`, কোনো নতুন dependency নেই।

**2026-10-20 — build visibility + asset path**

- Frontend assets moved `frontend/dist` → `frontend/web` (dist folders get
  stripped by archives/snapshots, which silently broke the embed).
- Build failures and GUI startup failures are now visible (log tail on screen,
  native error dialog, `startup-error.log`).
- All `.bat` files are CRLF. Deliverable zip now written to `release/`.
- Removed the duplicate root `eva_chat.zip`. No shipped binaries.
