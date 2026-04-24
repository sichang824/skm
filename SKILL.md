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
- 需要把 skill 附着到另一个 provider，或执行 link、move、sync、delete
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

在当前 skill 目录里创建或更新 `.to` 元数据：

```bash
skm skills to --provider-path ~/Workspace/skills
skm skills to --directory scripts --include README.md --include scripts/** --exclude assets/**
```

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
