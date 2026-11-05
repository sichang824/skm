# SKM

[中文文档](README.zh-CN.md)

**SKM is a local Skills manager built primarily for agents.** Give an agent one SKM entry point so it can search for capabilities, read the instructions it needs, and run declared commands. Maintain skill sources centrally and let multiple agents use them on demand.

[![CI & Release](https://github.com/sichang824/skm/actions/workflows/release.yml/badge.svg)](https://github.com/sichang824/skm/actions/workflows/release.yml)
[![Latest release](https://img.shields.io/github/v/release/sichang824/skm)](https://github.com/sichang824/skm/releases/latest)

**欢迎哔哩哔哩的小伙伴们使用 SKM！** [中文理念与使用说明](README.zh-CN.md) · [GitHub Issues](https://github.com/sichang824/skm/issues)

## Why I built SKM

I have more than 280 Skills on my computer, all serving agents. I built a GUI for SKM, but rarely open it: agents do most of the searching, reading, and invoking.

In my workflow, I do not install every business skill into each agent. SKM manages those skills, and agents learn one entry point for finding and using them. The size of the capability library can grow without requiring every task to carry the whole catalog.

**Keep capabilities available; bring their content into context when the task needs it.** Agents still need their operating instructions, permissions, and an SKM entry point. This approach moves infrequent, task-specific knowledge out of the default context. See [the author's design notes](docs/agent-first.md) (Chinese).

## Connect an agent

Install the CLI and make `skm` available to your agent (see [CLI](#cli)). Register and scan your skill source directories once. The CLI works without launching the GUI or web service.

```bash
# Replace this path with your skill source directory.
skm providers add --name "My Skills" --type workspace --root /path/to/skills
skm scan all

# Search when a task needs a skill.
skm skills -q meeting
skm skills --tag meeting

# Replace placeholders with a returned ZID and a declared command name.
skm skills get <skill-zid>
skm skills get <skill-zid> references/guide.md
skm skills get <skill-zid> --commands
skm skills exec <skill-zid> <command> --dry-run
skm skills exec <skill-zid> <command>
```

The keyword, tag, and file path above are examples; matches depend on your catalog. Search currently matches keywords in metadata such as names, directory names, tags, and summaries, rather than semantic similarity. Executable commands are optional.

A small bootstrap instruction for an agent can be:

> When a task needs specialized knowledge or tools, search with `skm skills -q <specific-keywords>`. Read a match with `skm skills get <ZID>`, then retrieve referenced files only as needed. Inspect declared commands and their arguments before invoking them with `skm skills exec`. Do not load the entire catalog or every skill body by default. Obtain user authorization when a command requires confirmation.

Alternatively, expose this repository's [SKILL.md](SKILL.md) as one skill entry point and read detailed usage when needed. Agents need local command access, access to the skill directories, and appropriate permissions. Remote or sandboxed agents need an accessible runtime or integration.

## What the implementation supports

- **Search before reading:** discover relevant skills at task time, then retrieve their instructions and individual files.
- **Shared sources:** multiple local agents can use the same indexed skill sources without copying the full business-skill collection into every agent.
- **Instructions connected to execution:** `SKILL.md` describes usage; `package.json` declares commands. `exec` handles arguments, input validation, environment checks, timeouts, and declared confirmation requirements.
- **Traceable calls:** execution records include status, exit codes, and content hashes. `--pin` can replay a historical version when matching content remains available.
- **People intervene where needed:** the GUI helps inspect sources, conflicts, and issues; agents can handle routine capability access while people set goals, authorize actions, and resolve exceptions.

## Context cost and future direction

Copying fewer files on disk does not by itself reduce tokens. The benefit comes from reducing irrelevant material entering the agent's context. Search results, loaded instructions, and command output still consume tokens; net savings depend on the workload and need measurement.

The existing [Agent Skills implementation guide](https://github.com/agentskills/agentskills/blob/main/docs/client-implementation/adding-skills-support.mdx) already describes loading skill bodies on demand. SKM's workflow also makes catalog discovery task-driven, so agents need not start with the entire catalog in context.

My expectation is that more information and capabilities will be consumed directly by agents, with more work delegated between them. SKM currently provides local skill discovery, reading, and execution; it does not implement the A2A protocol or a general event-stream platform. The design notes distinguish today's implementation from that future direction.

## Download

Get a ready-to-run app from **[GitHub Releases](https://github.com/sichang824/skm/releases/latest)**. Go and Node.js are only needed for development.

| Platform | Archive | Installation |
| --- | --- | --- |
| macOS (Apple Silicon and Intel) | `SKM-vVERSION-macos-universal.zip` | Extract and drag `SKM.app` into Applications |
| Windows 10/11 x64 | `SKM-vVERSION-windows-x64.zip` | Extract the entire archive and open `SKM.exe`; keep the adjacent `cli` folder |

For the optional GUI, add skill folders in **Providers**, then scan to browse and inspect them. Agents can use the CLI workflow above instead. Local data defaults to `.skm` in your user home directory and is preserved when replacing the app during upgrades.

Release builds are not developer-certificate signed or Apple-notarized. macOS may require “Open Anyway” in System Settings → Privacy & Security; Windows may show SmartScreen. Windows needs [Microsoft WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/); the app offers the official bootstrapper when it is missing (installation requires internet access).

Browsing and scanning need no extra shell tools. Running skill commands on Windows requires [Git for Windows](https://git-scm.com/download/win), with its `Git\bin` directory containing `sh.exe` on PATH. Install Node.js, Python, or other tools required by individual skill scripts separately.

## Preview

### Dashboard

![SKM dashboard](assets/dashboard.png)

The optional GUI gives people an inspection view of provider counts, skill totals, issues, scan activity, and provider distribution.

### Skills Catalog

![SKM skills catalog](assets/skills.png)

The skills catalog groups entries by provider, exposes related attached copies, and keeps browsing, filtering, and provider context in the same page.

## Design principles

Skill source directories, the index, and the invocation interface have distinct roles. Discovery is task-driven and content is loaded in stages. The CLI, REST API, and optional GUI share backend services. Attach, link, and sync remain available for compatibility; the preferred agent workflow uses centrally maintained sources with on-demand reading and execution.

See [the execution design](docs/skill-manifest-exec.md) for manifests, caching, version replay, and audit records. `--isolate` uses a cached working copy; it is not a permission sandbox.

## Architecture

```text
skm/
├── backend/    # Shared services, scanner, REST API and cmd/skm CLI
├── frontend/   # Optional React + Vite inspection UI
├── build/      # Desktop build output
├── main.go     # Wails desktop entry
├── wails.json  # Wails config
├── SKILL.md    # Agent entry instructions
└── README.md
```

## Tech Stack

- Backend: Go 1.24, Gin, GORM, SQLite or PostgreSQL
- Frontend: React 19, TypeScript, Vite, Tailwind CSS v4, React Router v7, Vitest
- Desktop: Wails v2.12.0

## Development

### Prerequisites

- Go 1.24+
- Node.js 22.12+
- pnpm 10+

### Run the full stack

```bash
make dev
```

Default endpoints:

- Backend: `http://localhost:8080`
- Frontend: `http://localhost:5173`

Use custom ports when needed:

```bash
BACKEND_PORT=18080 FRONTEND_PORT=4173 make dev
```

### Start with seeded providers

```bash
make dev/seed
```

This resets the local SQLite database, starts the backend and frontend, and seeds default provider records based on common skill directories found on the current machine.

### Run each part separately

Backend:

```bash
cd backend
cp .env.example .env
make run
```

Frontend:

```bash
cd frontend
make install
make dev
```

### Desktop development

```bash
make app-dev
```

### Build the macOS app

```bash
make app-build
```

The generated `.app` bundle is written to `build/bin/`.

If the desktop app is launched outside the repository working directory, SQLite resolves to `~/.skm/app.db` by default. Override it with `DB_DSN` when needed.

## Common Commands

```bash
make install       # Install frontend dependencies and preload Go modules
make dev           # Run backend and frontend together
make dev/seed      # Reset DB and start with seeded providers
make reset         # Remove local SQLite database files
make seed          # Seed default providers and exit
make test          # Run backend and frontend tests
make build         # Build backend binary and frontend assets
make cli-build     # Build the skm CLI binary
make cli-install   # Install the skm CLI to ~/.local/bin
make app-dev       # Run the Wails desktop app in dev mode
make app-build     # Build the macOS desktop app
```

## CLI

The CLI is the primary agent-facing entry point and shares the desktop app’s database and backend services. Use the CLI bundled in a Release (`SKM.app/Contents/Resources/skm` on macOS, `cli/skm.exe` on Windows) by putting it on PATH; launching the GUI is optional.

Build it:

```bash
make cli-build
```

Install it into `~/.local/bin`:

```bash
make cli-install
```

Commands:

```bash
skm version
skm dashboard
skm providers
skm providers add --name "Workspace Skills" --type workspace --root ~/Workspace/skills
skm providers update PROV0001 --priority 400 --description "main workspace provider"
skm providers delete PROV0001
skm skills --provider "Workspace Skills"
skm skills -q jira                       # skills that wrote this text in name, slug, directory, tags, or summary
skm skills --query "jira issue" --provider "Workspace Skills" --status ready   # combines with all filters
skm skills get SKIL0001                    # skill info + SKILL.md content + directory path
skm skills get SKIL0001 --files            # skill info + ls-style listing of all files
skm skills get SKIL0001 scripts/run.sh     # file info + file content (binary files show info only)
skm skills to --provider-path /path/to/provider --include README.md --include scripts/** --exclude plugins/**
skm skills link SKIL0001 --to PROV0002
skm skills move SKIL0001 --to PROV0003
skm skills sync SKIL0002
skm skills delete SKIL0002
skm issues --view latest
skm scan all
skm scan provider PROV0001
```

In the desktop app, the sidebar runtime card also exposes an `Install CLI` button. It installs the bundled CLI binary to `~/.local/bin/skm`.

## API Overview

Public endpoints:

```text
GET  /healthz
GET  /version
GET  /api/dashboard
```

Provider and scan endpoints:

```text
GET    /api/providers
POST   /api/providers
GET    /api/providers/:zid
PUT    /api/providers/:zid
DELETE /api/providers/:zid
POST   /api/providers/:zid/scan
POST   /api/scan
GET    /api/scan-jobs
GET    /api/scan-jobs/:zid
GET    /api/issues
GET    /api/issues?view=latest
GET    /api/conflicts
```

Skill endpoints:

```text
GET  /api/skills
GET  /api/skills?grouped=true
GET  /api/skills/:zid
POST /api/skills/:zid/attach
GET  /api/skills/:zid/files
GET  /api/skills/:zid/file-content?path=SKILL.md
```

See `docs/frontend-api-contract.md` for request and response examples.

## Configuration

Main runtime configuration lives in `backend/.env`:

- `PORT`: backend port, default `8080`
- `DB_DRIVER`: `sqlite` or `postgres`
- `DB_DSN`: database DSN, default `./data/app.db`
- `SEED`: seed default providers on startup
- `SEED_ONLY`: seed and exit

When running through Wails in development, the desktop host also reads `backend/.env`.

## Repository Guide

- Backend setup and API notes: `backend/README.md`
- Frontend setup and scripts: `frontend/README.md`
- Frontend API contract: `docs/frontend-api-contract.md`
- Product notes: `docs/PRD.md`

## Releases and version management

Versions are managed with **RustyTag**. Git tags are the source of truth; CI injects the tag into the app and CLI version. The repository configures the `v` prefix in `.rustytag.json`.

```bash
# Commit all product changes first, then create the next version on main.
rustytag show
rustytag patch        # use minor or major when appropriate
# For the first public release: rustytag patch --version v0.1.0

git push --follow-tags origin main
```

Only pushing a RustyTag-created `v*` version tag triggers this workflow. It runs tests, builds macOS universal and Windows x64 packages, then publishes a GitHub Release with both ZIP archives and `SHA256SUMS.txt` after all checks and builds succeed. Branch commits (including `main`), documentation updates, and pull requests do not trigger it. Tags containing a prerelease suffix are marked as prereleases. No personal access token is needed for CI publishing.

See [the release workflow guide](docs/releases.md) for local builds and release steps.

## Contributing

Issues and pull requests are welcome.

- Bug reports: include reproduction steps, expected behavior, actual behavior, and logs or screenshots when relevant
- Feature proposals: explain the user problem first, then the proposed interaction or API shape
- Pull requests: keep scope focused, update docs when behavior changes, and include validation steps

See `CONTRIBUTING.md` for the contribution workflow.

## Acknowledgements

SKM builds on top of excellent open source projects. Special thanks to:

- Wails for the desktop host and web-to-desktop integration model
- Gin and GORM for the Go-side HTTP and data access foundations
- React and Vite for the frontend architecture and development workflow
- Tailwind CSS, Radix UI, and shadcn/ui for UI building blocks
- Lucide, GSAP, Motion, and Zustand for icons, animation, and interaction state management

The project is also inspired by local-first toolchains and skill-directory workflows, especially the idea of using `SKILL.md` as a capability unit, directories as distribution units, and conflict visibility as a first-class feature.

## License

MIT. See `LICENSE`.
