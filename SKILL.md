---
name: skm
description: "Use when working with the local skm CLI to manage skill providers, catalog scans, issues, skill copies, and desktop workflow setup. Triggers: skm, providers, scan, skills link, skills move, skills sync, issues, dashboard."
---

# Skill: skm CLI

用 `skm` CLI 管理 skill provider、catalog、link/sync 副本与 issues。**Provider 是动态的**——运行时向 SKM 查 live catalog，不要背 ZID、路径或数量。

## 核心原则

1. **先理解意图，再动手** — 未弄清用户目标前，不要跑写操作。
2. **先查后改** — `dashboard` / `providers` / `skills` / `issues` 只读优先。
3. **动态解析 provider** — 用 `skm providers --json` 匹配；歧义时必须让用户选，不能默认。
4. **不写死 catalog** — skill 里不放 provider 清单、ZID 表、skill 数量。

## 第一步：理解意图

收到请求后，先归类（可并存），再选流程：

| 意图 | 典型说法 | 走哪条流 |
|------|---------|---------|
| 发布 / 附着 | 「link 到全局」「发到 Cursor」「挂到 workspace」 | [Publish 流程](#publish-流程) |
| 查询 / 核对 | 「有哪些 skill」「provider 对不对」 | 预检 + `providers` / `skills` |
| 刷新 catalog | 「扫一下」「列表没更新」 | `scan` |
| 修 `.to` / 副本 | 「同步副本」「只有 .from」 | `.to` 规则 + `sync-copies` |
| 排障 | 「CLI 没数据」「和 App 不一致」 | [预检](#预检cli-与-app-一致) + issues |
| 管理 provider | 「加一个 provider」「改 root」 | `providers add/update`（改前确认） |

**未明确 source / target provider、skill 名、或物理路径时** → 只读查询 + 向用户确认，不要猜。

## 预检：CLI 与 App 一致

任何「没数据 / 为空」结论前，先确认 CLI 连对库：

```bash
skm dashboard
skm providers --json
```

- App 有数据、CLI 为 0 → 查 `DB_DSN` / `DB_DRIVER`（常见：CWD 下无关 `.env` 污染；默认应为 `~/.skm/app.db` + `sqlite`）。
- 不要未验证 DSN 就断言「没有 provider / skill」。

## 动态解析 Provider

**一律** `skm providers --json`，按优先级匹配：

| 用户说法 | 匹配字段 |
|---------|---------|
| 明确名称（如 "Workspace Skills"） | `name` 精确或模糊 |
| 明确路径 | `rootPath` |
| 模糊词（「全局」「cursor」「workspace」） | 多命中 → **列候选让用户选** |

解析到 provider 后，口头或回复中确认：`name`、`rootPath`、zid（来自查询结果，非 skill 记忆）。

## Publish 流程

把 skill **从 source provider 发布到 target provider**（link 副本）时，按序执行：

**0. 预检** — `skm dashboard` / `skm providers --json`（见上）

**1. 解析 target** — 动态查 provider；歧义则问用户

**2. 确认 source 与物理位置**

- 技能须在 **source provider 的 `rootPath` 下**，且为**真实目录**（不要用 symlink 当源）
- 不在 → 用户要求 `mv` 就 `mv` 进去；未要求 relocate 则先问
- 不要擅自 symlink 代替 mv

**3. 刷新 catalog**

```bash
skm scan provider <source-provider-zid>
skm skills --provider "<source-name>" --query <skill-name>
```

**4. Link + 同步**

```bash
skm skills link <source-skill-zid> --to <target-provider-zid>
skm skills sync-copies <source-skill-zid>
```

**5. 验证**

```bash
skm skills get <source-skill-zid>    # relation targets 正确
ls <target-rootPath>/<skill-dir>/    # 文件齐全，不只是 .from
```

link 后只有 `.from` → 源必须是真实目录，补 `.to` 规则后再 `sync-copies`。

## 不变量

- **源目录**：真实目录，位于 source provider `rootPath` 内。
- **Provider**：运行时查 catalog；名称/路径模糊时禁止默认。
- **破坏性操作**：`move`、`delete`、`providers delete` 仅在用户明确要求时执行；执行前展示目标与影响。
- **CLI/App**：dashboard 为空先查 DSN，再下结论。
- **JSON**：需要结构化分析时用 `--json`。
- **扫描**：列表 stale、冲突不对 → 先 `scan provider` 或 `scan all`。
- **副本同步**：源更新后 `sync-copies <source-zid>`；单副本用 `skills sync <copy-zid>`。

## `.to` 元数据

`.to` 定义 link/sync 时复制到消费端的文件子集。**手写只填 `include` / `exclude`；`directories` 由 link 写入。**

| 通常 `include` | 通常 `exclude` |
|----------------|----------------|
| `SKILL.md` | 源码（`internal/**`、`cmd/**`、`src/**`） |
| `scripts/**`、`references/**` | 测试、构建产物（`bin/**`、`dist/**`、`node_modules/**`） |
| `assets/**`、运行时需要的 `Makefile` | 用户数据、日志、密钥（`.env`、`media/**`、`output/**`） |

```bash
cd path/to/skill
skm skills to --include SKILL.md --include 'scripts/**' --exclude 'media/**'
```

改 `.to` 后：`scan provider` → 对已附着副本 `skills sync` 或 `sync-copies`。

## 命令速查

```bash
skm version | dashboard | providers [--json] | skills [--json] | issues | scan all | scan provider <zid>
skm skills get <zid> | skills link <zid> --to <prov-zid> | skills sync <zid> | skills sync-copies <zid>
skm providers add | update | delete    # delete 需用户明确授权
skm skills move | delete               # 破坏性，需用户明确授权
```

CLI 未安装：在 skm 仓库执行 `make cli-install`。开发栈：`make dev` / `make dev/seed`。

## 排障顺序

1. `skm version` + `skm dashboard`
2. `skm providers --json` — 空则查 DSN
3. `skm scan all` 或 `scan provider <zid>`
4. `skm issues`
5. `skm skills get <zid>`

## 何时不用这个 skill

- 改 `skm` 源码、Wails 宿主、前端 API 调试（非 CLI 操作流）。
- 仅需 `make dev` / `make test` / `make app-build` 等仓库开发命令。
