# SKM Frontend API Contract

本文档覆盖当前前端已经接入的页面：Dashboard、技能列表页、技能详情弹窗、Provider 页、异常与冲突页。

## 基础约定

- 基础路径：`/api`
- 响应结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

- 前端开发环境默认通过 Vite 代理把 `/api` 转发到 `http://localhost:8080`
- 如果不用代理，可设置环境变量：`VITE_API_BASE_URL=http://localhost:8080`

## 页面与接口映射

### 技能列表页

- `GET /api/dashboard`
- `GET /api/providers`
- `GET /api/skills?q=&provider=&status=&sort=`

### Dashboard

- `GET /api/dashboard`
- `GET /api/providers`
- `GET /api/skills?sort=lastScanned`
- `GET /api/issues?view=latest`
- `GET /api/scan-jobs`

列表页当前使用的 `sort` 枚举：

- `name`
- `provider`
- `status`
- `lastScanned`

### 技能详情页

- `GET /api/skills/:zid`
- `GET /api/skills/:zid/files`
- `GET /api/skills/:zid/file-content?path=SKILL.md`
- `GET /api/skills/:zid/commands`（可执行命令面板；`skill.commands.length > 0` 时展示入口）
- `POST /api/skills/:zid/exec`（执行命令；confirm 门禁未放行 → 409，预检失败 → 422，命令本身失败是 200 + exitCode）
- `GET /api/execs?skill=<zid>&limit=10`（执行历史；HASH 可复制为 `--pin` 值）

### Provider 页

- `GET /api/providers`
- `POST /api/providers`
- `PUT /api/providers/:zid`
- `DELETE /api/providers/:zid`
- `POST /api/providers/:zid/scan`
- `POST /api/scan`
- `GET /api/skills?sort=provider`
- `GET /api/issues?view=latest`

### 异常与冲突页

- `GET /api/issues?view=latest`
- `GET /api/conflicts`

## 关键对象

### Provider

```json
{
  "zid": "PROV000000000001",
  "name": "skills-workspace",
  "type": "workspace",
  "rootPath": "/path/to/skills",
  "enabled": true,
  "priority": 100,
  "scanMode": "recursive",
  "description": "主技能工作区",
  "lastScannedAt": "2026-04-15T02:41:56Z",
  "lastScanStatus": "completed",
  "lastScanSummary": "added=3 removed=0 changed=1 invalid=1 conflicts=1"
}
```

### Skill

```json
{
  "zid": "SKIL000000000123",
  "name": "OpenAPI Inspector",
  "slug": "openapi-inspector",
  "directoryName": "openapi-inspector",
  "rootPath": "/path/to/skills/openapi-inspector",
  "skillMdPath": "/path/to/skills/openapi-inspector/SKILL.md",
  "category": "integration",
  "tags": ["openapi", "contract"],
  "summary": "Review and call documented endpoints.",
  "status": "ready",
  "contentHash": "8b6e...",
  "lastModifiedAt": "2026-04-15T02:41:56Z",
  "lastScannedAt": "2026-04-15T02:42:01Z",
  "bodyMarkdown": "# OpenAPI Inspector\n...",
  "frontmatter": {
    "name": "OpenAPI Inspector",
    "category": "integration",
    "tags": ["openapi", "contract"]
  },
  "issueCodes": [],
  "conflictKinds": ["name_content_diff"],
  "isConflict": true,
  "isEffective": true,
  "provider": {
    "zid": "PROV000000000001",
    "name": "skills-workspace"
  }
}
```

### Latest Issue

`GET /api/issues?view=latest` 返回的是“每个 Provider 最近一次扫描”上的问题集合，并按 `(provider, skill, rootPath, relativePath, code, severity, message)` 去重。

```json
{
  "zid": "SISS000000000045",
  "code": "name_directory_mismatch",
  "severity": "warning",
  "message": "directory name does not match skill name",
  "rootPath": "/path/to/skills/wrong-folder",
  "relativePath": "SKILL.md",
  "createdAt": "2026-04-15T02:42:01Z",
  "provider": {
    "zid": "PROV000000000001",
    "name": "skills-workspace"
  },
  "skill": {
    "zid": "SKIL000000000124",
    "name": "OpenAPI Inspector"
  }
}
```

### Skill Exec（`POST /api/skills/:zid/exec`）

请求体（`SkillExecPayload`）：

```json
{
  "command": "transcribe",
  "args": ["media/meeting.m4a"],
  "input": "{\"k\":\"v\"}",
  "env": ["API_KEY=xxx"],
  "assumeYes": false,
  "timeoutSeconds": 0,
  "isolate": false,
  "pin": "3fa2c1d8",
  "dryRun": false
}
```

- `pin`（可选）：sourceHash 前缀（8–64 hex），按历史版本执行，只在缓存副本里跑；留空跑最新
- 响应为 `SkillExecResult`：`ok/exitCode/timedOut/aborted/workDir/durationMs/stdout/stderr`，另含 `deps`（托管安装：node/python 命令行、ran/skipped、exitCode/durationMs）、`setup`（同形状）、`plan`（dry-run 展示用：`pin/deps/depsSkipped/setupSkipped/commandLine`）
- HTTP 语义：预检失败（未知命令、缺 env、confirm 409、input/pin 无效 422）走错误码；命令本身失败是 200 + `exitCode`

### ExecRecord（`GET /api/execs`）

执行审计记录（新→旧），`skill` 过滤、`limit` 上限 200：

```json
{
  "zid": "EXEC000000000007",
  "skillZid": "SKIL000000000124",
  "skillName": "tingwu-transcribe",
  "command": "transcribe",
  "trigger": "http",
  "who": "ann",
  "workDir": "/path/to/skills/tingwu-transcribe",
  "mode": "source",
  "pin": "",
  "sourceHash": "3fa2c1d8…（64 hex，可用作 --pin）",
  "status": "completed",
  "exitCode": 0,
  "timedOut": false,
  "reason": "",
  "args": ["media/meeting.m4a"],
  "envKeys": ["TINGWU_API_KEY"],
  "inputVia": "",
  "durationMs": 1234,
  "startedAt": "2026-08-19T02:42:01Z"
}
```

- `status`：`completed | failed | timeout | setup_failed | deps_failed | rejected`；`rejected` 行带 `reason`
- 机密面：只存 env 键名，不存值；input 不落库

## 扫描规则

### 默认扫描模式

- 新建 Provider 默认使用 `scanMode=recursive`
- `recursive` 会递归查找包含 `SKILL.md` 的目录，并把该目录视为 skill 根目录
- `shallow` 只扫描 Provider 根目录下的一级子目录

### 冲突分类

- `name_duplicate`: 同名且内容哈希一致
- `name_content_diff`: 同名但内容哈希不一致
- `path_duplicate`: 同一路径冲突

## 示例请求

### 1. 拉技能列表页数据

```bash
curl 'http://localhost:8080/api/dashboard'
curl 'http://localhost:8080/api/providers'
curl 'http://localhost:8080/api/skills?sort=lastScanned&status=ready'
```

### 2. 拉技能详情页数据

```bash
curl 'http://localhost:8080/api/skills/SKIL000000000123'
curl 'http://localhost:8080/api/skills/SKIL000000000123/files'
curl 'http://localhost:8080/api/skills/SKIL000000000123/file-content?path=SKILL.md'
```

### 3. 新建 Provider 并触发扫描

```bash
curl -X POST 'http://localhost:8080/api/providers' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "skills-workspace",
    "type": "workspace",
    "rootPath": "/path/to/skills",
    "enabled": true,
    "priority": 100,
    "scanMode": "recursive",
    "description": "主技能工作区"
  }'

curl -X POST 'http://localhost:8080/api/providers/PROV000000000001/scan'
```

### 4. 拉 Provider 页联调数据

```bash
curl 'http://localhost:8080/api/providers'
curl 'http://localhost:8080/api/skills?sort=provider'
curl 'http://localhost:8080/api/issues?view=latest'
```

### 5. 触发全量扫描并查看 Dashboard / Issues

```bash
curl -X POST 'http://localhost:8080/api/scan'
curl 'http://localhost:8080/api/scan-jobs'
curl 'http://localhost:8080/api/conflicts'
```
