# Skill 可执行清单（package.json）与 `skills exec` 设计

状态：v1.1（第 8 期落地后更新）｜ 日期：2026-08-19

## 1. 背景与目标

当前 skill 分发依赖 link 物理副本（41 个副本，24 个含可执行脚本）。`skm skills get` 已支持按需读取 skill 内容，但含脚本的 skill 无法脱离物理副本——脚本必须落盘才能执行。

本设计为 skill 引入**构建期声明的可执行清单**（复用 `package.json`），由 skm 解析并辅助执行，使 HAS-CODE skill 也能走「按需获取」模式，最终消除 link 副本。

**目标**

1. skill 在 `package.json` 中声明可执行命令；skm scan 时解析、入 catalog、可校验
2. `skm skills exec` 提供统一的「解析 → 预检 → 执行」通道：参数传递、env 校验、确认门禁、超时、结构化输出
3. 执行白名单化：只有声明过的命令可被 skm 执行
4. 为删除 link 副本铺路：exec 直接在源目录运行，无副本也能用

**非目标**

- 不提供任意 shell 透传（`exec <zid> -- <任意命令>` 不存在）
- 不替代 SKILL.md（SKILL.md 仍是 agent 的指令入口，Operations 章节由清单生成）
- 不做沙箱隔离（脚本以用户权限运行；清单治理的是「调用面」，不是权限边界）
- 依赖安装托管只做 opt-in 的标准安装动作（`skm.runtime.deps`，§2.3）；不做跨 skill 共享的安装缓存，脚本额外的安装需求仍由 setup 命令自理

## 2. Manifest：package.json

复用 npm 生态约定：`scripts` 是**命令注册表**（单一事实源），`skm` 扩展节是**执行元数据注解**。

### 2.1 完整示例（bash 脚本类，tingwu-transcribe 形态）

```json
{
  "name": "tingwu-transcribe",
  "version": "1.2.0",
  "description": "转写本地音频为文字（听悟 API）",
  "scripts": {
    "setup": "bash scripts/setup.sh",
    "transcribe": "bash scripts/transcribe.sh",
    "list-media": "bash scripts/list_media.sh"
  },
  "skm": {
    "schemaVersion": 1,
    "runtime": {
      "env": ["TINGWU_API_KEY"],
      "setup": "setup"
    },
    "commands": {
      "transcribe": {
        "description": "转写一个本地音频文件并导出 markdown",
        "timeoutSeconds": 1800
      },
      "list-media": {
        "description": "列出 media/ 下的音频文件"
      }
    }
  }
}
```

### 2.2 完整示例（结构化输入类，Shiju operations 形态）

```json
{
  "name": "reimbursement-assistant",
  "version": "0.1.18",
  "scripts": {
    "claims:delete": "node operations.js claims.delete",
    "claims:list": "node operations.js claims.list"
  },
  "skm": {
    "schemaVersion": 1,
    "runtime": { "env": ["SHIJU_KEY"] },
    "commands": {
      "claims:delete": {
        "description": "Delete a reimbursement claim",
        "confirm": "将删除报销单，不可恢复",
        "input": {
          "via": "argv",
          "schema": {
            "type": "object",
            "properties": { "claimId": { "type": "string", "minLength": 1 } },
            "required": ["claimId"]
          }
        }
      },
      "claims:list": {
        "description": "List reimbursement claims",
        "input": {
          "via": "stdin",
          "schema": {
            "type": "object",
            "properties": {
              "status": { "enum": ["draft", "submitted", "approved", "rejected"] }
            }
          }
        }
      }
    }
  }
}
```

### 2.3 字段定义

| 字段 | 必填 | 说明 |
|------|------|------|
| `name` / `version` / `description` | 推荐 | npm 标准字段；`name` 应与 SKILL.md frontmatter `name` 一致，不一致产 issue（warning） |
| `scripts.<cmd>` | 是 | 命令行（shell 字符串）。**唯一**可以声明「执行什么」的地方；npm 兼容，`npm run <cmd>` 可独立使用 |
| `skm.schemaVersion` | 是（有 skm 节时） | 当前为 `1` |
| `skm.runtime.env` | 否 | skill 级必需环境变量**名**（不含值），exec 前统一校验 |
| `skm.runtime.setup` | 否 | 首次执行前要跑一次的命令名（指向 scripts 某条），如装 venv、拉模型 |
| `skm.runtime.deps` | 否 | 依赖安装托管开关（**opt-in**，见下）；键存在即开启 |
| `skm.commands.<cmd>` | 否 | 对同名 scripts 命令的注解；**纯增量**，不存在对应 scripts 条目 → scan issue |
| `commands.<cmd>.description` | 推荐 | 一行说明；进入 catalog，供发现与文档生成 |
| `commands.<cmd>.confirm` | 否 | `true` 或自定义警示文案。exec 必须带 `--yes` 才放行 |
| `commands.<cmd>.timeoutSeconds` | 否 | 超时强杀（杀进程组）；未声明 = 不限时 |
| `commands.<cmd>.env` | 否 | 该命令额外必需的 env 名，与 runtime.env 合并校验 |
| `commands.<cmd>.input` | 否 | 结构化输入：`{ via, schema }`，见 §3 |

**依赖安装托管（`skm.runtime.deps`）**

显式声明才启用，现有 skill 行为零变化：

```json
"runtime": {
  "deps": {}
}
```

- `{}`：自动探测两个生态——工作目录有 `package-lock.json`（或 package.json 声明了 dependencies/devDependencies）→ node；有 `requirements.txt` → python
- `{"node": true}` / `{"python": true}`：按生态显式选择
- 安装动作：node 有 lockfile 跑 `npm ci --no-audit --no-fund`，无 lockfile 跑 `npm install …`；python 先按需 `python3 -m venv .venv`，再 `.venv/bin/pip install -r requirements.txt`
- 幂等：完成标记 `.skm-deps.json` 按「工作目录 × lockfile/requirements 内容哈希」记账（与 setup 标记同址同模式），声明不变不重装；重物化缓存副本会连标记一起清掉，隔离副本因此各自重装
- 执行时 `.venv/bin` 与 `node_modules/.bin`（存在时）前置进 PATH（命令走 `sh -c`，需要与 npm-run 等价的可见性）
- 安装失败 = `Aborted="deps-failed"`，与 setup 失败同形：命令不启动
- 未知键（如 `"nodee": true`）→ scan issue `manifest_deps_unknown`（error），防拼写错误静默关闭托管

**设计原则**

1. `scripts` 单一事实源；`skm.commands` 只做注解，两者按名字 merge
2. 只在 `scripts` 声明、无 `skm` 注解 → 完全合法，exec 以默认参数运行（通用兜底）
3. 全增量：现有 skill 零改动；已有 package.json 的 node skill 只加 `skm` 节

## 3. 参数传递：三条通道

设计目标：**零声明也能跑，强约束也能管**。

### 通道 A：位置参数（永远可用，零声明）

```bash
skm skills exec <zid> transcribe -- media/meeting.m4a --dir-id 21469 --title "周会"
```

`--` 之后的参数逐个 shell-quote 后追加到 scripts 命令行尾（等价 `npm run transcribe -- ...`）。工作目录 = skill 根目录，所以相对路径引用（`scripts/x.sh`、`media/`）自然成立。这是通用兜底：任何脚本不改 manifest 就能接收 argv。

### 通道 B：结构化输入（可选声明，执行前校验）

```bash
skm skills exec <zid> claims:delete --input '{"claimId":"C-123"}'
skm skills exec <zid> claims:list  --input @query.json
cat query.json | skm skills exec <zid> claims:list --input -
```

- `input.schema`：JSON Schema。skm 在执行前校验，不合 schema 直接拒绝（错误信息指出违反的字段），不启动脚本
- `input.via` 声明投递方式：

| via | 投递 | 适用 |
|-----|------|------|
| `stdin`（默认） | JSON 写入进程 stdin | 新写脚本的首选，无 ARG_MAX 问题 |
| `argv` | 作为单个追加参数（在 `-- args` 之前） | 兼容 Shiju 现状 `npm run x -- '<json>'` |
| `env` | 注入环境变量 `SKM_INPUT` | 只能读 env 的运行时 |

- 通道 A、B 可组合：`--input` 与 `-- args` 同时给，投递顺序为先 input（argv 模式）后位置参数

### 通道 C：环境变量

- 声明式校验：`runtime.env` / `commands.x.env` 中的变量缺失 → **拒绝执行**并列出缺什么（不让它死在脚本深处）
- 显式注入：`--env KEY=VAL` 可重复
- 当前进程环境照常透传；**skm 不自动加载任何 `.env`**（防密钥意外进入执行环境；脚本要加载可自行 source）

### skm 注入的标准变量

脚本可选用，无需声明：

| 变量 | 含义 |
|------|------|
| `SKM_SKILL_ROOT` | 执行目录（skill 根，绝对路径） |
| `SKM_SKILL_ZID` / `SKM_SKILL_NAME` / `SKM_SKILL_VERSION` | catalog 身份 |
| `SKM_COMMAND` | 本次执行的命令名 |
| `SKM_INPUT` | `input.via=env` 时的 JSON |

## 4. 执行位置：源目录优先，缓存兜底

这是让 link 副本真正消失的关键决策：

1. **源目录存在 → 直接在源目录执行**（skill 记录的 `rootPath`，真实目录）
   - 脚本相对引用、可写状态目录（`media/`、`output/`）与 link 时代行为一致
   - 若 exec 的 zid 是 link 副本，沿 relation 解析到其 source 执行；source 不可用才退回副本目录
2. **无本地目录**（未来 remote/hub skill）→ 按 `.to` 规则物化到 `~/.skm/cache/exec/<zid>/`；目录内写 `.skm-cache.json`（sourceZid、sourcePath、contentHash、sourceHash、materializedAt）。contentHash + sourceHash（按拷贝规则对源树做的内容哈希）未变复用，变化重物化；无 `.to` 时拷贝全部内容（排除 node_modules/.git 等垃圾目录），package.json、package-lock.json、requirements.txt 永远强制包含
3. `--isolate`：强制走缓存物化执行，保护源目录不被写脏
4. **runtime.setup 幂等**：完成标记 `.skm-setup.json` 也存在缓存目录（源目录模式同样如此，源目录不落 skm 状态），按「工作目录 × manifest 哈希」记账——源目录与缓存副本各自独立，manifest 变化才重跑；命令执行前自动触发（顺序：依赖安装 → setup），失败则中止命令（不启动脚本）
5. **依赖安装幂等**：`skm.runtime.deps` 声明开启后，安装完成标记 `.skm-deps.json` 与 setup 标记同址，按「工作目录 × lockfile/requirements 哈希」记账（§2.3）
6. **并发保护（flock）**：缓存目录的兄弟文件 `<cacheRoot>/<zid>.lock` 承载 advisory 锁（锁在缓存目录内会被重物化的 RemoveAll 删掉）。物化/依赖安装/setup 的 check-run-write 临界区持**独占锁**（新鲜度检查在锁内重做，双检），源目录消失的兜底读取持**共享锁**；命令执行期间不持锁。dry-run 不写盘也不抢锁
7. **版本 pin（`--pin <hash前缀>`）**：按物化副本的 sourceHash 前缀（8–64 hex）选版本执行——缓存副本的 sourceHash 命中 → 直接复用；源树当前哈希命中 → 重物化**同一个**缓存目录后使用；都不命中 → 报错（版本不可恢复，可经 `skills execs` 查历史哈希）。pinned 执行**永不在源目录跑**（即使源哈希命中也物化到缓存），且按设计绕过 contentHash 相等判断。单目录保守策略：不做历史版本归档，磁盘不随 pin 增长

## 5. `skills exec` 命令面

```
skm skills exec <skill-zid> <command> [flags] [-- args...]

  --input <json|@file|->   结构化输入（B 通道）
  --env KEY=VAL            env 注入，可重复（C 通道）
  --yes                    confirm 命令的放行条件
  --timeout <sec>          覆盖 manifest 超时
  --dry-run                打印解析结果（cmd / cwd / env / input 投递），不执行；
                           不实际运行，因此不受 confirm 门禁拦截
  --json                   结构化输出
  --isolate                强制物化到缓存执行
  --pin <hash>             按 sourceHash 前缀（8–64 hex）选版本执行（§4.7），
                           只在缓存副本里跑；可发现哈希见 skills execs
  --setup                  只执行 runtime.setup（幂等标记由 skm 维护）；
                           声明了 deps 时先装依赖；可与 --pin 组合
  --force                  配合 --setup：即使完成标记有效也重跑
```

**执行流程**（顺序固定）：

1. 解析 skill zid → catalog 记录 → 校验 pin 格式 → 读 `package.json`（不存在 → 报错「该 skill 未声明可执行清单」）
2. 查命令：必须在 `scripts` 中；未命中 → 报错并列出可用命令（含模糊建议）
3. 预检：env 必填校验 → input schema 校验 → confirm 门禁（无 `--yes` → 非零退出，附清单里的警示文案）
4. 解析执行目录（§4，pin 优先）；实际执行前按需自动安装声明的依赖，再执行 `runtime.setup`（两者各自幂等；任一失败都中止命令）
5. 执行：`sh -c "<script line>"` + 追加参数，cwd = 执行目录；stdout/stderr 流式透传；工作目录存在 `.venv/bin` / `node_modules/.bin` 时前置进 PATH
6. 退出码透传脚本退出码；超时杀进程组，退出码 124
7. 审计：除 dry-run 外，每次调用（含预检拒绝）落一条 `exec_records` 记录（§5.1）

**`--json` 输出包**：

```json
{
  "ok": false,
  "exitCode": 2,
  "timedOut": false,
  "skill": { "zid": "...", "name": "tingwu-transcribe" },
  "command": "transcribe",
  "workDir": "/Users/ann/Workspace/skills/tingwu-transcribe",
  "durationMs": 1234,
  "stdout": "...",
  "stderr": "..."
}
```

交互终端流式输出；`--json` 时缓冲后一次性输出（供 agent 机器消费）。

### 5.1 exec 审计：`skills execs`

除 dry-run 外，每次调用（含预检拒绝、setup/deps 中止）由单一收口落一条 `exec_records` 记录：skill 身份、命令、trigger（cli/http）、执行者、workDir、mode、pin、**sourceHash**（本次运行内容的树哈希，即可被 `--pin` 重放的版本）、状态（completed/failed/timeout/setup_failed/deps_failed/rejected）、退出码、耗时、参数与 env **键名**（机密面：env 值与 input 不落库）。表无外键：删除 skill/provider 后审计记录保留。落库 best-effort，绝不影响执行本身。

```bash
skm skills execs [--skill <zid>] [--limit N] [--json]
```

HASH 列展示 sourceHash 前 12 位——这是发现可 pin 版本的入口；App 端 `GET /api/execs?skill=&limit=` 提供同一数据（执行历史面板 + hash 复制）。

## 6. 安全模型

| 机制 | 说明 |
|------|------|
| 白名单 | 只有 `scripts` 声明的命令可执行；无任意命令透传形态 |
| confirm 由 skm 强制 | 替代 Shiju「脚本自律」（APP_OPERATION_CONFIRMED），脚本不再自己实现确认逻辑 |
| 预检失败不启动进程 | env 缺失、schema 不过、confirm 未放行，都在进程启动前拒绝 |
| 不读 .env | skm 不自动加载任何密钥文件 |
| 明确边界 | 清单治理调用面，不是沙箱：声明的命令仍以用户全权限运行 |

## 7. scan 集成与 catalog 索引

scan 发现 SKILL.md 同目录的 `package.json` 时：

**解析与入库**
- Skill 模型新增 `Commands` 字段（serializer:json）：`[{name, description, line, confirm, timeoutSeconds, hasInput}]`
- 有 `scripts` → skill 具备「可执行」属性（App UI badge、`skills --executable` 过滤，后续）

**新增 issue 码**

| code | 级别 | 条件 |
|------|------|------|
| `manifest_invalid_json` | error | package.json 解析失败 |
| `manifest_command_missing` | error | `skm.commands` 条目无对应 `scripts` |
| `manifest_deps_unknown` | error | `skm.runtime.deps` 键未知或非 bool（生态名拼错会静默关闭托管） |
| `manifest_target_missing` | warning | 命令行引用的相对路径文件不存在（轻量启发式：`scripts/` 路径 token） |
| `manifest_name_mismatch` | warning | package.json `name` ≠ SKILL.md frontmatter `name` |

**`skills get <zid> --commands`**：列出命令清单（名称、描述、confirm、env 要求、input 方式），agent 发现能力用。

## 8. SKILL.md 生成：单一事实源

Operations 章节由 manifest 构建期生成（每条命令：描述 + 调用模板 + confirm 提示），避免 manifest 与散文双写漂移。Shiju 现有生成管线（OpenAPI → SKILL.md operations）已是此形态，泛化复用。

生成的调用模板统一指向 exec：

```markdown
### transcribe

转写一个本地音频文件并导出 markdown

Command: `skm skills exec <skill-zid> transcribe -- <file> [--dir-id <id>] [--title <title>]`
```

不生成的 skill 不受影响（agent 照旧读 SKILL.md 手动调脚本）。

## 9. 兼容与迁移

**兼容性**

- 纯增量：无 package.json 的 skill 一切照旧
- npm 兼容：`scripts` 即 npm scripts，node 生态 skill 无缝
- link/sync 机制不变；manifest 只影响 exec 与 scan 校验

**副本迁移路径**（对应 24 个 HAS-CODE link 副本）

1. 为源 skill 补 `package.json`（bash 类通常只是包一层 `scripts` + 少量注解）
2. `skm skills exec <zid> <cmd> --dry-run` 逐一验证解析正确
3. 真实调用验证等价后，删除对应 link 副本；副本位置 agent 的 SKILL.md 调用说明改为 exec
4. docs-only 副本按此前结论直接走 fetch（`skills get`），不需要 manifest

## 10. 实现分解（建议顺序）

| # | 内容 | 位置 | 状态 |
|---|------|------|------|
| 1 | manifest 解析 + Skill.Commands 字段 + scan issue 码 + 测试 | `internal/models`、`internal/service/manifest.go`、`scan_service.go` | ✅ 2026-08-18 |
| 2 | `skills get --commands` | CLI（`backend/cmd/skm`） | ✅ 2026-08-18 |
| 3 | exec 引擎：解析 → 预检 → 执行（源目录模式、流式输出、超时、退出码） | `internal/service/exec_service.go`、`jsonschema.go` | ✅ 2026-08-18 |
| 4 | `skills exec` 全部 flags（--input/--env/--yes/--dry-run/--json/--timeout） | CLI | ✅ 2026-08-18 |
| 5 | 物化缓存 + `--isolate` + runtime.setup 幂等 | exec 引擎扩展（`internal/service/exec_cache.go`） | ✅ 2026-08-18 |
| 6 | SKILL.md operations 生成工具 / 泛化 Shiju 管线 | `internal/service/operations.go` + `skills gen-operations` | ✅ 2026-08-18 |
| 7 | App 端：可执行 badge、命令列表、Web 触发 exec | handlers（`GET /api/skills/:zid/commands`、`POST /api/skills/:zid/exec`）+ frontend | ✅ 2026-08-18 |
| 8 | exec 审计入 DB：`exec_records` 单一收口 + `skills execs` + `GET /api/execs` + App 执行历史 | `models/exec_record.go`、`service/exec_audit.go`、CLI、handlers、frontend | ✅ 2026-08-19 |
| 9 | 缓存物化/安装 flock（`<zid>.lock`，独占/共享） | `service/exec_cache.go` | ✅ 2026-08-19 |
| 10 | 依赖安装托管（opt-in `skm.runtime.deps`，npm/venv+pip，`.skm-deps.json` 幂等） | `service/manifest.go`、`service/exec_deps.go`、`exec_service.go` | ✅ 2026-08-19 |
| 11 | 版本 pin：`--pin` sourceHash 前缀，单缓存目录，pin 不入源目录 | `service/exec_cache.go`、`exec_service.go`、CLI/HTTP/frontend | ✅ 2026-08-19 |

## 11. Open Questions

1. ~~**依赖安装**~~：✅ 已落地（第 8 期，2026-08-19）——opt-in `skm.runtime.deps` 托管安装：skm 感知 `package-lock.json`/`requirements.txt`，跑 `npm ci`/`npm install` 与 venv+pip，按 lockfile 哈希幂等；不做跨 skill 共享安装缓存（node_modules/.venv 留在各自工作目录，不进物化副本）。后续可选项：安装输出缓存、超时策略
2. ~~**并发**~~：✅ 已落地（第 8 期，2026-08-19）——`<cacheRoot>/<zid>.lock` advisory flock：物化/依赖安装/setup 的 check-run-write 持独占锁（锁内复查新鲜度），源消失兜底读取持共享锁；命令执行期间不持锁。源目录模式天然无物化竞争，setup/deps 标记经同一把锁串行
3. ~~**exec 审计**~~：✅ 已落地（第 8 期，2026-08-19）——`exec_records` 表记录每次非 dry-run 调用（含预检拒绝），字段见 §5.1；`skills execs` / `GET /api/execs` / App 执行历史
4. **平台**：`sh -c` 语义假定 POSIX shell；Windows 支持策略待定（进程组杀、flock、venv 路径均为 POSIX 实现）
5. ~~**版本 pin**~~：✅ 已落地（第 8 期，2026-08-19）——`--pin <sourceHash前缀>`（8–64 hex）：缓存副本命中 → 复用；源树哈希命中 → 重物化同一缓存目录；都不中 → 报错。单目录保守策略，不归档历史版本；可 pin 哈希经审计表发现（§5.1）
