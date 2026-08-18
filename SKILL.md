---
name: skm
description: "Use when working with the local skm CLI to manage skill providers, catalog scans, issues, skill copies, on-demand skill content retrieval (skills get), and package.json command execution (skills exec). Triggers: skm, providers, scan, skills link, skills move, skills sync, skills get, skills exec, execs, gen-operations, package.json commands, issues, dashboard."
---

# Skill: skm CLI

用 `skm` CLI 管理 skill provider、catalog、link/sync 副本与 issues。**Provider 是动态的**——运行时向 SKM 查 live catalog，不要背 ZID、路径或数量。

## 核心原则

1. **先理解意图，再动手** — 未弄清用户目标前，不要跑写操作。
2. **先查后改** — `dashboard` / `providers` / `skills` / `issues` 只读优先。
3. **动态解析 provider** — 用 `skm providers --json` 匹配；歧义时必须让用户选，不能默认。
4. **不写死 catalog** — skill 里不放 provider 清单、ZID 表、skill 数量。
5. **内容按需取** — 消费端默认只持 `SKILL.md` 副本做发现，正文/脚本/执行一律走 `skills get` / `skills exec`，不要再整包 link 副本。

## 第一步：理解意图

收到请求后，先归类（可并存），再选流程：

| 意图 | 典型说法 | 走哪条流 |
|------|---------|---------|
| 取用内容 / 执行 | 「看这个 skill 的内容」「读它的脚本」「跑 skill 里的 xx 命令」 | [get 取用](#直接取用-skill-内容get) / exec |
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
skm dashboard          # 先看 Database: 行指向哪个库
skm providers --json
```

- App 有数据、CLI 为 0 → 查 `DB_DSN` / `DB_DRIVER`（默认应为 `~/.skm/app.db` + `sqlite`）。
- **在 skm 仓库内（尤其 `backend/`）跑 CLI 会被 `backend/.env` 的 `DB_DSN=./data/app.db` 指到 dev 库**——看起来「provider 变少 / 数据不见了」时，先换到干净 CWD（如 `/tmp`）重跑再下结论。
- 不要未验证 DSN 就断言「没有 provider / skill」。

## 动态解析 Provider

**一律** `skm providers --json`，按优先级匹配：

| 用户说法 | 匹配字段 |
|---------|---------|
| 明确名称（如 "Workspace Skills"） | `name` 精确或模糊 |
| 明确路径 | `rootPath` |
| 模糊词（「全局」「cursor」「workspace」） | 多命中 → **列候选让用户选** |

解析到 provider 后，口头或回复中确认：`name`、`rootPath`、zid（来自查询结果，非 skill 记忆）。

## 直接取用 skill 内容（get）

现行主用法：**agent 端只留 SKILL.md 副本做发现，全部内容按需取**（link 副本已清零，`.to` 的 include 有意只留 `SKILL.md`）。不需要把 skill 整包同步进各 agent home。

三种形态：

```bash
skm skills get <zid>            # 信息 + SKILL.md 全文（磁盘优先、永远最新；磁盘缺失才退回 catalog 快照并注明）+ 目录路径
skm skills get <zid> --files    # ls -l 风格文件清单（权限/大小/时间/相对路径）
skm skills get <zid> <path>     # 读单个文件：文本直接打印内容；二进制只显示信息不打印；目录列子项（get <zid> . 看根目录）
```

- 路径安全：`../` 越界被拒（exit 1）；绝对路径在 skill 内自动转相对。
- 每个文本视图末尾带 `Next:` 推荐命令（真实 zid/path 已填好，可直接复制执行）——顺着提示逐级深入：概览 → `--files` → 具体文件。
- `--files`/path 与 `--json` 均可用；`--files` 与 path 混用报错（exit 2）。
- 发现可执行命令后接 `skills exec`（见下节）。

## Publish 流程

把 skill **从 source provider 发布到 target provider**（link 副本）时，按序执行。**注意现行约定：副本默认只含 `SKILL.md`（供发现），内容靠 get/exec 按需取**——仅当消费端接不上 skm 时才同步整包。

**0. 预检** — `skm dashboard` / `skm providers --json`（见上）

**1. 解析 target** — 动态查 provider；歧义则问用户

**2. 确认 source 与物理位置**

- 技能须在 **source provider 的 `rootPath` 下**，且为**真实目录**（不要用 symlink 当源）
- 不在 → 用户要求 `mv` 就 `mv` 进去；未要求 relocate 则先问
- 不要擅自 symlink 代替 mv

**3. 刷新 catalog**

```bash
skm scan provider <source-provider-zid>
skm skills --provider "<source-name>" -q <skill-name>   # -q/--query 模糊搜索（多词 AND，按相关度排序），可与 --category/--tag/--status/--conflict/--sort 组合
```

`-q` 匹配规则：完全匹配 > 前缀 > 子串 > 子序列（`brauth` 能命中 `browserauth`），大小写不敏感，中文可用；字段加权 name > tags > slug/目录名 > category/provider > summary。

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
| `SKILL.md`（现行约定：agent 发现只需它，其余走 get/exec） | 源码（`internal/**`、`cmd/**`、`src/**`） |
| `scripts/**`、`references/**`（仅当消费端接不上 skm 时整包） | 测试、构建产物（`bin/**`、`dist/**`、`node_modules/**`） |
| `assets/**`、运行时需要的 `Makefile` | 用户数据、日志、密钥（`.env`、`media/**`、`output/**`） |

```bash
cd path/to/skill
skm skills to --include SKILL.md --include 'scripts/**' --exclude 'media/**'
```

CLI 语义（重要）：

- `--directory` 是**追加**目标目录；`--include` / `--exclude` 是**整组替换**。
- `--provider-path <path>` 复用或创建 provider root（必须是当前目录或其父目录）。
- 清理陈旧条目时记得：`directories` 里指向已删目录的条目会让 `sync-copies` 复活副本，需一并清掉。

改 `.to` 后：`scan provider` → 对已附着副本 `skills sync` 或 `sync-copies`。

## 可执行清单（package.json）与 exec

skill 可在目录里放 `package.json` 声明可执行命令：`scripts` 是命令注册表（npm 兼容），`skm` 节是注解（`confirm` / `timeoutSeconds` / `env` / `input.{via,schema}`）。scan 时解析入库（`Skill.Commands`），校验失败产 `manifest_*` issue。

```bash
skm skills get <zid> --commands                        # 查看声明的命令
skm skills exec <zid> <command> [-- args...]           # 执行（只有 scripts 声明过的命令可跑）
skm skills exec <zid> <command> --input '{"k":"v"}'    # 结构化输入，按 schema 校验后投递（stdin/argv/env）
skm skills exec <zid> <command> --input @in.json       # --input 三种形态：内联 JSON | @文件 | -（stdin）
skm skills exec <zid> <command> --env K=V              # 注入环境变量（可重复）
skm skills exec <zid> <command> --timeout 60           # 覆盖 manifest 的 timeoutSeconds
skm skills exec <zid> <command> --yes                  # confirm 命令必须带 --yes
skm skills exec <zid> <command> --dry-run              # 预览解析结果，不执行
skm skills exec <zid> <command> --isolate              # 在缓存副本（~/.skm/cache/exec/<zid>/）里跑，不写脏源目录
skm skills exec <zid> <command> --pin <hash前缀>       # 按 sourceHash 前缀（8-64 hex）跑历史版本（只在缓存副本跑）
skm skills exec <zid> --setup [--isolate] [--force] [--pin hash] [--dry-run]   # 只跑 runtime.setup（幂等；标记由 skm 维护）
skm skills execs [--skill <zid>] [--limit N] [--json]  # 执行审计历史（默认 20 条；状态/退出码/HASH，HASH 可用作 --pin）
skm skills gen-operations <zid> [--check] | --all      # 从 package.json 重生成 SKILL.md 的 Operations 章节
```

要点：**白名单执行**（无任意命令透传）；confirm 由 skm 强制；env 缺了在进程启动前就拒绝；link 副本自动解析到源目录执行；退出码透传（超时 124）。首次执行前按需自动装依赖（仅 manifest 声明 `skm.runtime.deps` 的 skill：npm ci/install、venv+pip，按 lockfile 哈希幂等）再跑 `runtime.setup`（按「工作目录 × manifest 哈希」幂等；两者失败都不启动命令）。`--isolate` 按 `.to` 规则物化缓存副本执行（物化/安装/setup 有 flock 防并发）；源目录不在时，contentHash 未变的缓存仍可兜底。`--pin` 按 `skills execs` 里的 sourceHash 前缀重放历史版本：缓存命中即复用，源树哈希命中则重物化，都不中报错。每次非 dry-run 调用都落审计记录（含预检拒绝；只存 env 键名不存值，input 不落库）。`gen-operations` 让 SKILL.md 的命令说明与 manifest 保持单一事实源（生成块带标记，可重复执行；`--check` 供 CI 校验漂移）。设计文档：`docs/skill-manifest-exec.md`。

## 命令速查

```bash
skm version | dashboard | providers [--json] | skills [--json] | issues | scan all | scan provider <zid>
skm skills [-q text] [--provider zid-or-name] [--category v] [--tag v] [--status v] [--sort name|provider|status|lastScanned] [--conflict true|false]
skm skills get <zid> [path] [-f/--files] [--commands] | skills link <zid> --to <prov-zid> | skills sync <zid> | skills sync-copies <zid>
skm skills exec <zid> <command> [--input json|@file|-] [--env K=V] [--yes] [--timeout sec] [--isolate] [--pin hash] [--dry-run] [--json] [-- args...]
skm skills exec <zid> --setup [--isolate] [--force] [--pin hash] | skills execs [--skill <zid>] [--limit N] | skills gen-operations <zid>|--all [--check]
skm skills to [--directory <path> ...] [--include <glob> ...] [--exclude <glob> ...] [--provider-path <path>]
skm issues [-code x] [-severity x] [-provider zid-or-name] [-view latest|all] [-json]
skm providers add --name n --type t --root p [--scan-mode recursive|shallow] [--priority N] [--icon name] [--description text] [--enabled=true]
skm providers update <zid> [同 add 的可选 flag] | providers delete <zid>    # delete 需用户明确授权，且不接受 --json
skm skills move <zid> --to <prov-zid> | skills delete <zid> [--force]        # 破坏性，需用户明确授权；--force = 有副本也删源
```

CLI 未安装：在 skm 仓库执行 `make cli-install`。开发栈：`make dev` / `make dev/seed`。

## 排障顺序

1. `skm version` + `skm dashboard`（确认 `Database:` 行）
2. `skm providers --json` — 空则查 DSN（仓库内 CWD 污染见预检）
3. `skm scan all` 或 `scan provider <zid>`
4. `skm issues`（按 `-code`/`-severity` 过滤）
5. `skm skills get <zid>`

## 何时不用这个 skill

- 改 `skm` 源码、Wails 宿主、前端 API 调试（非 CLI 操作流）。
- 仅需 `make dev` / `make test` / `make app-build` 等仓库开发命令。
