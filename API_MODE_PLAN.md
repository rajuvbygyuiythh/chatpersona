# API mode split — plan & exact file scope

**Status:** waiting for owner approval (no code changed yet)
**Date:** 2026-10-20
**Scope:** dashboard `Message` section (shared by Show Mode and Background Bot)
**Goal:** `EVAChat Flow` and `API` become two separate chat-mode options instead of
AI being a checkbox inside EVAChat Flow.

---

## 1. How it works today (measured, not guessed)

The Message panel has exactly three modes:

| # | Mode | Value | Panel |
|---|---|---|---|
| 1 | Single Sequence | `single` | textarea |
| 2 | Folder Sequences | `folder` | Load Folder / Clean Loaded |
| 3 | EVAChat Flow | `eva_chat_flow` | Open data folder + **AI reply text (hybrid)** block |

The "API option" the owner sees is a **sub-toggle of EVAChat Flow**, not a mode:

- `cmd/chitchatui/frontend/web/index.html:219-247` — the AI block lives inside
  `#sm-section-evachat-mode` (and `#bg-section-evachat-mode` at 490+).
- `internal/model/ui.go:34-52` — `MessageSettings` carries `AIEnabled`,
  `AIBaseURL`, `AIAPIKey`, `AIModel`.
- `app.go:872-925` — `buildBotConfigs()` only creates an `EVAChatConfig` when
  `EffectiveChatMode() == eva_chat_flow`; `AI` is a field of that struct
  (`internal/bot/eva_chat.go:169`).
- `internal/bot/eva_chat.go:1157-1174` — `evaChatStart/Reply` call
  `evaChatCompose()` which rewrites the TXT line (`internal/bot/eva_ai.go:431`).
- `app.go:681` — `TestEVAChatAI()` powers the "Test AI" button.
- Show Mode never used the AI layer: `live_chat.go:178-200` builds an
  `evaLiveResponder` from the TXT engine only.

So there is **one** `chat_mode` switch with an AI sub-feature. That is why the
two things cannot be selected independently today.

---

## 2. Target UX after the change

```
Chat mode:  ( ) Single Sequence   ( ) Folder Sequences
            ( ) EVAChat Flow      ( ) API Chat        <- two separate choices

EVAChat Flow panel                       API Chat panel (new)
----------------------------------       -------------------------------------
[Open EVAChat Data Folder]               [Open API Chat Data Folder]
TXT rules decide flow.                   API base URL  https://api.featherless.ai/v1
(no API fields here)                     Model         deepseek-ai/DeepSeek-V3-0324
                                         API key       ••••••••   [Test API]
                                         Temperature   0.9
                                         Share at reply number:
                                            HOT 4   WARM 6   COLD 8
Snap / Telegram users  (shared field, unchanged)
```

- **EVAChat Flow** = today's behaviour with AI switched off: pure TXT engine,
  deterministic stages, no network calls.
- **API Chat** = the whole conversation is written by the model, driven by the
  funnel design already proven in `apitest/` (interest tier → mood → plan →
  share at reply 4/6/8 → guards → end). TXT files are not used in this mode
  except a small editable `fallback.txt` deck used only when the API fails.

Both pages (Show Mode + Background Bot) get the same 4 modes and the same
shared store.

---

## 3. Engine design for API Chat

### 3.1 New package `internal/apichat` (stdlib only)

A production port of `apitest/` (keep `apitest/` untouched as the tuning tool).
Stdlib-only keeps it testable in isolation and avoids new dependencies.

| File | Responsibility | Ported from |
|---|---|---|
| `director.go` | interest scoring, HOT/WARM/COLD tier, moods, plan (instruction/double/share/end/fixed), memory extraction (age, gender, place, name, doing), guards (under-18, insult, money/OF, non-English, spam, no-snap, added, bye), share-turn decision | `apitest/main.go:169-770` |
| `client.go` | OpenAI-compatible `/chat/completions`, timeout, thinking-off fields for known reasoning models, error classification, one retry | `apitest/main.go:1233-1352` + existing `eva_ai.go:348` |
| `prompt.go` | persona template (`{age} {country} {name} {major} {year} {tonight} {show} {snack}`), history window, director note as last system message | `apitest/main.go:804-826`, `persona.txt` |
| `sanitize.go` | one line only, lowercase, max 10 words, no `@`/links/handles, no bot tells, no echo/parrot, question-policy retry | `apitest/main.go:1403-1560` |
| `share.go` | share deck with `%username%`, hook line before share, share/end sequencing | `apitest/main.go:160-166`, `474-710` |
| `session.go` | JSON-serialisable `Session` (transcript, director state, persona age/country, counters) | new |
| `engine.go` | `Start(ctx)` / `Reply(ctx, text)` → `Result{Lines, Rule, Detail, HasUsername, Ended, Shared}`, fallback deck on API error | new glue |

**Hard rules carried over:** username is appended by Go, never written by the
model; under-18 → polite end; no explicit content; English only; never repeat a
line; max 10 words per line; max 2 lines.

### 3.2 Runtime data — new folder `data/api_chat/`

Seeded from embedded assets (`internal/bot/api_assets/`), same "never overwrite
user edits" logic as `EnsureEVAChatData`:

```
data/api_chat/
├── persona.txt            # system prompt (seeded with the apitest persona)
├── persona_countries.txt  # bot's own country pool for {country}
├── share.txt              # share lines containing %username%
├── fallback.txt           # short generic lines used only if the API fails
└── README.txt
```

### 3.3 State & persistence

`model.AccountState` gains `APIChatSessions map[string]apichat.Session`
(transcript capped at 30 lines, like EVA's 24). Saved in the existing
`bot_state.json`, so a restart continues the same conversation.

### 3.4 Fallback behaviour (no silent bot)

| Failure | Action |
|---|---|
| API 4xx/5xx, timeout (25 s), bad JSON | log once, send a line from `fallback.txt`, keep flow alive |
| Model output fails a guard | 1 retry, then fallback line |
| Missing API key in API mode | start is refused with a clear message (like Folder mode without sequences) |

---

## 4. Settings, migration, compatibility

`internal/model/ui.go`

```go
ChatModeAPI = "api_chat"          // new 4th mode

type MessageSettings struct {
    // ...existing fields...
    APIBaseURL     string  `json:"api_base_url,omitempty"`
    APIAPIKey      string  `json:"api_api_key,omitempty"`
    APIModel       string  `json:"api_model,omitempty"`
    APITemperature float64 `json:"api_temperature,omitempty"`
    APIShareHot    int     `json:"api_share_hot,omitempty"`   // default 4
    APIShareWarm   int     `json:"api_share_warm,omitempty"`  // default 6
    APIShareCold   int     `json:"api_share_cold,omitempty"`  // default 8
    // legacy ai_* fields stay readable for migration only
}
```

- `EffectiveChatMode()` returns `api_chat` for the new value.
- On load, if `api_*` is empty and legacy `ai_*` exists → copy (the owner does
  not have to retype the key). Legacy `ai_enabled` is ignored.
- API key is stored only in the local `data/` settings (same as today); it is
  never logged, never sent to the UI after save (masked), never committed.

---

## 5. Exact file scope

### 5.1 NEW files (9 + tests + assets)

| File | Est. lines | Purpose |
|---|---:|---|
| `source_code/internal/apichat/director.go` | 430 | funnel brain (port) |
| `source_code/internal/apichat/client.go` | 130 | API transport + retry |
| `source_code/internal/apichat/prompt.go` | 110 | persona + director note |
| `source_code/internal/apichat/sanitize.go` | 160 | output guards |
| `source_code/internal/apichat/share.go` | 120 | share deck + sequencing |
| `source_code/internal/apichat/session.go` | 70 | persisted state |
| `source_code/internal/apichat/engine.go` | 180 | Start/Reply/fallback |
| `source_code/internal/apichat/*_test.go` (3-4 files) | 380 | unit tests (fake HTTP server) |
| `source_code/internal/bot/api_chat.go` | 150 | worker glue (`apiChatStart/Reply/Render`, enable check) |
| `source_code/internal/bot/api_chat_test.go` | 220 | glue tests |
| `source_code/internal/bot/api_assets.go` | 80 | embed + `EnsureAPIChatData` |
| `source_code/internal/bot/api_assets/{README,persona,persona_countries,share,fallback}.txt` | 120 | seeded editable pack |
| `source_code/docs/API_CHAT_MODE.md` | 120 | how to use/tune |

### 5.2 MODIFIED files

| File | Anchor | Change |
|---|---|---|
| `internal/model/ui.go` | 22-52, ~195 | new mode const, new fields, `EffectiveChatMode` |
| `internal/model/model.go` | 229-240 | `APIChatSessions` field |
| `internal/uistore/store.go` | 81-124 | save/load + legacy `ai_*` migration |
| `internal/uistore/store_test.go` | — | migration test |
| `internal/bot/run.go` | 40-70 | `Config.APIChat *APIChatConfig` |
| `internal/bot/logic.go` | 385-395, 476-495, 712-722 | 3 dispatch points → `apiChatEnabled` branch |
| `internal/bot/logic.go` | 1029-1031, after 1198 | match dispatch + `executeMatchAPIChat` (~130) |
| `internal/bot/eva_chat.go` | 1142-1180 | remove the 2 `evaChatCompose` calls (pure TXT) |
| `internal/bot/eva_ai.go` + `eva_ai_test.go` | whole | **retired** — client/sanitize logic moves into `internal/apichat` |
| `live_chat.go` | 96-200 | `apiLiveResponder` + mode branch in `newLiveResponder` |
| `live_chat_test.go` | — | responder test |
| `app.go` | 395, 662, 681, 872-925 | save/validate mode, `OpenAPIChatDataFolder`, `TestAPIChat`, config build |
| `cmd/.../dist/index.html` | 183-256, 411-490 | 4th radio + API panel on both pages; remove AI block from EVA panel |
| `cmd/.../dist/app.js` | 373-470, actions | 4-mode switch, payload, folder/test actions, key validation |
| `cmd/.../dist/styles.css` | — | ~15 lines (4-option row / API grid) |
| `cmd/.../wailsjs/go/main/App.js` / `App.d.ts` | — | `OpenAPIChatDataFolder`, `TestAPIChat` |
| `tests/contracts.test.cjs` | 45-75 | mode list + new IDs + fallback contract |
| `tests/browser-ui.test.cjs` | 139+, 237 | 4-mode UI tests |
| `AGENTS.md`, `docs/PROJECT_MEMORY.md` | — | decision log entry |
| `docs/PROJECT_MAP.md` | AUTO | regenerated by `npm run map:update` |

### 5.3 Explicitly UNTOUCHED

`apitest/**` (tuning tool stays as is) · `capture/**` · `internal/proxy`,
`internal/browser`, `internal/api`, `internal/license` · Single & Folder modes ·
`internal/bot/eva_assets/**` (TXT pack) · root `install.bat`,
`run-dashboard.bat` · `package.json` (no new dependency).

---

## 6. Task phases (each phase ends with a check)

| Phase | Work | Done when |
|---|---|---|
| **0** | model + store + config fields + migration | old settings load unchanged; legacy key migrates |
| **1** | `internal/apichat` package + tests | director/sanitize/share unit tests pass logically (see §7) |
| **2** | bot glue + 3 dispatch points + match path + live responder | API mode produces opener/reply/share/end per plan |
| **3** | app layer: data seeding, folder RPC, test RPC, config build | dashboard can open `data/api_chat` and test the key |
| **4** | dashboard: 4 modes, API panel, bindings | both pages show 4 options; values survive tab switches |
| **5** | tests, docs, `npm run map:update`, clean zip | `npm test` green + zip ≤ 700 KB |

---

## 7. Verification — what is and is not possible here

**Can run in this sandbox (I will run all of them):**

- `node --check` on every changed JS
- `npm test` (jsdom UI + contract tests, currently 72 → expected ~85)
- `gofmt` (wasm) on every changed Go file → no formatting drift
- tree-sitter Go parse on the whole tree → no syntax errors
- zip sanity: no `data/`, `node_modules/`, `.exe`, logs; required files present

**Cannot run here (no Go toolchain, module proxy blocked):**

- `go build`, `go vet`, `go test`, `wails build`

This is a real limit and it is why the plan keeps the new engine in a
stdlib-only package with a small, explicit surface. First real compile happens
on the owner's machine with `install.bat`. If the compiler reports anything,
paste the exact error text and I fix it in the next zip — no data loss, the
runtime `data/` folder is never touched by a rebuild.

---

## 8. Risks and mitigations

| Risk | Mitigation |
|---|---|
| compile error in glue code (cannot be caught here) | small API surface, mirror existing EVA patterns, gofmt + parse gate, error text → quick fix |
| API latency inside match chat | 25 s timeout, 1 retry, fallback deck, existing reply-delay logic respected |
| model writes a username / link | sanitizer strips, `share.txt` is the only source of the handle |
| token cost per reply | max 60 tokens, 24-line history, one call per bot line |
| `bot_state.json` growth | transcript capped at 30 lines per user |
| owner edits persona wrongly | `README.txt` in the data folder + `Test API` button |

---

## 9. Decisions needed before coding

1. **API Chat brain** — pure API funnel (recommended, apitest design) or the
   current hybrid (TXT picks the line, AI rewrites it)?
2. **Old AI block inside EVAChat Flow** — remove it (recommended) or keep it
   hidden/off by default?
3. **Data folder** — new `data/api_chat/` (recommended) or reuse
   `data/eva_chat/`?
4. **Show Mode** — API Chat should work there too (recommended, same as
   EVAChat Flow) or Background Bot only?
