# oprov — opencode provider & key manager

A terminal UI for managing **custom providers, models, and API keys** in your
OpenCode configuration. It looks and behaves like OpenCode's own TUI because it
is built on the same stack (Bubble Tea, Lip Gloss, Huh) and uses the exact
color tokens from OpenCode's default `opencode` theme.

```
● oprov · opencode providers                                  ~/.config/opencode/opencode.jsonc  jsonc
╭──────────────────────────────╮╭───────────────────────────────────────────────────────────────╮
│ PROVIDERS                4  ││ retard                                                          │
│ ─────────────────────────── ││ ─────────────────────────────────────────────────────────────  │
│   retard          1 model(s)││ id        mercury                                                │
│   mercury                ○  ││ name      retard                                                 │
│   dummier          1 model(s)││ package   @opencode/ai/providers/openai-compatible             │
│ ❯ guzz             2 model(s)││ base url  https://api.inceptionlabs.ai/v1                       │
│   hyper                  ·  ││ api key   ○ {env:MY_CUSTOM_API_KEY}  (unset)                     │
│   duce                   ·  ││ env       MY_CUSTOM_API_KEY                                      │
│                              ││                                                               │
│                              ││ MODELS  2                                                      │
│                              ││   ❯ deepseek-ai/DeepSeek-V4-Flash-0731   deepseek  400k ctx     │
│                              ││     mercury-3                               mercury             │
│                              ││                                                               │
│                              ││                                                               │
├──────────────────────────────┴───────────────────────────────────────────────────────────────┤
│ 4 provider(s) · ~/.config/opencode/opencode.jsonc                    ? help · q quit          │
│ [a] add  [e] edit  [d] del  [c] key  [x] clear  [t] test  [tab] detail  [q] quit              │
╰───────────────────────────────────────────────────────────────────────────────────────────────╯
```

## What it manages

* **Providers** — add / edit / delete entries under `providers` in your global
  `opencode.json(c)` (or any config you point it at): display name, runtime
  package, base URL, env-var list.
* **Models** — add / edit / delete models per provider: model id / alias,
  display name, context & output limits, input/output cost per 1M tokens,
  tool + reasoning support, disable flag.
* **API keys** — set, change, clear and verify keys, with three storage modes:
  * **Protected file (recommended)** — writes the secret to
    `<config-dir>/keys/<provider>.key` (mode `0600`) and stores a `{file:…}`
    reference in the config. Works even when the background OpenCode service
    is started without your shell environment.
  * **Environment variable** — stores `{env:VAR}` (and adds it to the
    provider's `env` list). If the variable isn't exported yet, it offers to
    append the `export` line to your shell rc file.
  * **Inline** — writes the key directly into the config (warns that it's in
    plaintext).
* **Test connection** — issues `GET <baseURL>/models` with the resolved key and
  reports the result (e.g. `connected · 4 model(s) listed`, `authentication
  failed (401)`, or a friendly network error).

Key status is shown in the provider list and detail pane:

`●` green = resolvable now · `○` amber = configured but not resolvable · `·` dim = no key

## Install

```sh
git clone https://github.com/yourname/oprov && cd oprov
make build          # or: go build -o oprov .
# put ./oprov on your PATH, e.g.
sudo install -m755 oprov /usr/local/bin/oprov
```

Requires Go 1.26+.

## Usage

```sh
oprov                          # manage the global config (~/.config/opencode/opencode.jsonc)
oprov --path path/to/config.jsonc   # manage a specific config (project files work too)
```

Every change is written to disk immediately; OpenCode watches config files and
picks changes up without a restart.

## Keys

| key | list focused                  | detail focused                  |
| --- | ----------------------------- | ------------------------------- |
| `j`/`k`/`↑`/`↓` | move selection        | select model                    |
| `g` / `G`       | first / last          | first / last model              |
| `a`             | add provider          | add model                       |
| `e` / `enter`   | edit provider         | edit model                      |
| `d`             | delete provider*      | delete model*                   |
| `c`             | set / change API key  | set / change API key            |
| `x`             | clear API key*        | clear API key*                  |
| `t`             | test connection       | test connection                 |
| `tab`/`enter`   | focus detail pane     | —                               |
| `esc` / `h`     | —                     | back to provider list           |
| `?`             | toggle keybinds help  |                                 |
| `q` / `ctrl+c`  | quit                  |                                 |

\* destructive actions ask for confirmation (`y`/`n`).

Inside forms (add/edit popups): `tab`/`enter` moves down, `↑` (and `←` in
dropdowns) moves up, `esc` closes without saving.

## Design

The palette mirrors OpenCode's default `opencode` TUI theme (v2.2.x):

| token      | value   |
| ---------- | ------- |
| background | `#0a0a0a` |
| panel      | `#141414` |
| border     | `#484848` |
| text       | `#eeeeee` |
| muted      | `#808080` |
| accent     | `#9d7cd8` (purple) |
| interactive| `#fab283` (orange) |

Form fields, focus states, selection markers, dialogs and toasts follow the
same component language as the OpenCode TUI.

## How it stores things

* The config file is edited as a **JSONC syntax tree** (via Tailscale's
  hujson): comments, trailing commas, key order and unrelated settings are all
  preserved across edits.
* Secrets never appear in the config except in the `inline` mode (which warns).
  File-based keys live in `keys/` next to the config with `0600` permissions.
* Writes are atomic (temp file + rename).

## Layout

```
internal/config   config loading, JSONC editing, key resolution, connection test
internal/ui       Bubble Tea app: panes, dialogs, forms, theme, status bar
```
