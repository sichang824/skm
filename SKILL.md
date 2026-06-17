---
name: skm
description: "Use when working with the local skm CLI to manage skill providers, catalog scans, issues, skill copies, and desktop workflow setup. Triggers: skm, providers, scan, skills link, skills move, skills sync, issues, dashboard."
---

# Skill: skm CLI

使用这个 skill 来操作本地 `skm` CLI，完成 skill provider 管理、目录扫描、技能检索、附着副本同步、问题排查，以及桌面工作流相关的基础操作。

## 适用场景

- 用户提到 `skm`、skill provider、catalog scan、issues、dashboard
- 需要新增、更新、删除 provider
- 需要扫描一个或全部 provider
- 需要查询 skill、查看 skill 详情、维护 `.to` 元数据
- 需要把 skill 附着到另一个 provider，或执行 link、move、sync、sync-copies、delete
- 需要先用 CLI 核对数据，再去桌面端或前端界面继续操作

## 基本约束

- 删除 provider 或 skill 前，默认先展示目标对象并确认影响范围。
- `move`、`delete`、`providers delete` 属于破坏性操作；如果用户没有明确要求，不要主动执行。
- 需要脚本化消费结果时，优先使用 `--json`。
- 当 `issues`、`skills`、`providers` 的参数不明确时，先跑只读查询，避免直接修改状态。
- 如果本机没有 `skm` 命令，优先在仓库根目录执行 `make cli-install` 或 `make cli-build`。

## 前置条件

1. `skm` CLI 可执行文件已安装，或可在仓库内构建：

```bash
make cli-build
make cli-install
```

1. 若从源码仓库启动完整环境，可使用：

```bash
make dev
```

1. 若要重置并带种子数据启动，可使用：

```bash
make dev/seed
```

## 命令总览

顶层命令：

```bash
skm dashboard
skm issues
skm providers
skm scan
skm skills
skm version
```

需要查看帮助时：

```bash
skm --help
skm providers --help
skm skills --help
skm scan --help
```

## 推荐工作流

### 1. 检查 CLI 与总体状态

先确认 CLI 可用、版本正确，再看 dashboard。

```bash
skm version
skm dashboard
```

如果需要机器可读输出，优先查看子命令是否支持 `--json`。

### 2. 管理 providers

列出当前 providers：

```bash
skm providers
skm providers --json
```

新增 provider：

```bash
skm providers add \
  --name "Workspace Skills" \
  --type workspace \
  --root ~/Workspace/skills
```

可选参数包括：`--scan-mode recursive|shallow`、`--enabled`、`--priority`、`--icon`、`--description`。

更新 provider：

```bash
skm providers update PROV0001 --priority 400 --description "main workspace provider"
```

删除 provider：

```bash
skm providers delete PROV0001
```

### 3. 执行扫描

扫描全部 providers：

```bash
skm scan all
```

扫描单个 provider：

```bash
skm scan provider PROV0001
```

当用户反馈目录内容没同步、skill 列表不更新、issues 结果过旧时，优先先做一次 scan。

### 4. 查询和管理 skills

按条件列出 skills：

```bash
skm skills
skm skills --provider "Workspace Skills"
skm skills --query prompt --sort lastScanned
skm skills --conflict true
skm skills --json
```

查看单个 skill：

```bash
skm skills get SKIL0001
```

### 维护 `.to` 元数据

`.to` 是 skill 目录里的 JSON 文件，定义 **attach / sync 时要复制到消费端（如 `~/.cursor/skills/...`）的文件子集**。

**原则：包含 skill 被调用时需要的全部内容；排除源码、构建产物和运行时垃圾。**

| 通常应 `include` | 通常应 `exclude` |
|------------------|------------------|
| `SKILL.md` | 语言源码（如 `internal/**`、`cmd/**`、`*.go`、`src/**`） |
| `scripts/**`、可执行包装脚本 | 测试（`**/*_test.*`、`**/*.test.*`、`testdata/**`） |
| `references/**`（OpenAPI、API 说明、配置模板） | 构建输出（`bin/**`、`dist/**`、`node_modules/**`） |
| `assets/**`（skill 运行时引用的静态资源） | 用户数据、缓存、日志、cookie（`media/**`、`output/**`、`*.log`） |
| `Makefile`（若 skill 文档里会 `make ...`） | 密钥、本地配置（`.env`、`*_secret*`） |
| 预编译 CLI 二进制（若 skill 直接调用且仓库内分发） | `.git/**`、`.to`、`.from`、编辑器目录 |

`directories` 由 `skm skills link` / sync 流程写入，表示已附着副本的目标路径。**手写 `.to` 时只维护 `include` / `exclude`，不要填 `directories`。**

文件格式示例（仅规则，无目标路径）：

```json
{
  "include": [
    "SKILL.md",
    "Makefile",
    "scripts/**",
    "references/**"
  ],
  "exclude": [
    "**/.DS_Store",
    "media/**",
    "output/**"
  ]
}
```

用 CLI 创建或更新（在 skill 目录内执行；`--include` / `--exclude` 可重复）：

```bash
cd path/to/skill
skm skills to \
  --include SKILL.md \
  --include Makefile \
  --include 'scripts/**' \
  --include 'references/**' \
  --exclude 'media/**' \
  --exclude 'output/**'
```

也可直接编辑 `.to` 后执行 `skm scan provider <zid>` 刷新 catalog。`link` / `sync` 会按 `.to` 规则复制文件；改规则后对已附着副本执行 `skm skills sync <zid>`。

**对照示例**

- Shell-only skill（如 `tingwu-transcribe`）：`SKILL.md` + `scripts/**` + `references/**`，排除用户音频与输出。
- CLI skill 仓库（如 `openapi`）：`SKILL.md` + `assets/**` + `bin/**` + `scripts/**`，排除 Go 源码与测试。

创建附着副本：

```bash
skm skills link SKIL0001 --to PROV0002
```

移动 skill 到另一个 provider：

```bash
skm skills move SKIL0001 --to PROV0003
```

同步附着副本：

```bash
skm skills sync SKIL0002
```

从关联源同步到全部关联副本：

```bash
skm skills sync-copies SKIL0001
```

删除 skill：

```bash
skm skills delete SKIL0002
skm skills delete SKIL0002 --force
```

### 5. 查看 catalog issues

`issues` 没有子命令，直接通过 flags 过滤：

```bash
skm issues
skm issues -view latest
skm issues -view all -severity error
skm issues -provider PROV0001 -code CONFLICT_SKILL_NAME
skm issues -json
```

当用户想定位冲突、扫描异常、目录结构问题时，先查看 issues，再决定是否补扫或调整 provider。

## Agent 操作建议

- 先查后改：先用 `providers`、`skills`、`issues` 的只读命令拿到上下文，再执行写操作。
- 改动前锁定对象：对 `update`、`delete`、`link`、`move`、`sync`，先确认目标 `zid` 是否正确。
- 维护 `.to` 时以「agent 执行 skill 所需最小文件集」为准：漏 include 会导致附着副本缺脚本或 spec；误 include 源码会膨胀副本且无助于调用。
- 关联源更新后，用 `skm skills sync-copies <source-zid>` 或 UI「同步全部副本」推送至所有 `.to` 目录；单个副本仍用 `skm skills sync <copy-zid>`。
- 扫描优先：遇到“界面没刷新”“skill 丢失”“冲突状态不对”时，优先尝试 `skm scan all` 或 `skm scan provider`。
- JSON 优先：当后续需要结构化分析或二次处理时，使用支持 `--json` 的命令。
- 破坏性操作最小化：除非用户明确要求，否则不要直接执行 `move`、`delete` 这类不可逆或高影响命令。

## 常见排障顺序

1. `skm version`，确认 CLI 存在且版本正常。
2. `skm providers`，确认 provider 已注册且根目录正确。
3. `skm scan all` 或 `skm scan provider <zid>`，刷新 catalog。
4. `skm issues`，查看是否有冲突、缺失、扫描异常。
5. `skm skills` / `skm skills get <zid>`，确认目标 skill 当前状态。

## 何时不用这个 skill

- 任务是修改 `skm` 源码实现本身，而不是使用 CLI。
- 任务重点是前端页面、Wails 桌面宿主或后端 API 调试，此时应优先查看对应代码和测试。
- 用户只需要仓库开发命令，如 `make dev`、`make test`、`make app-build`，而不涉及 CLI 操作流。
