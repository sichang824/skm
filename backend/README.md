# SKM Backend

SKM 后端负责管理 Provider、扫描本地技能目录、建立索引、识别异常与冲突，并向前端或桌面宿主提供统一 REST API。

## Responsibilities

- Provider 的增删改查、启用停用和优先级管理
- 单 Provider 扫描与全量扫描
- 技能元数据提取，包括 `SKILL.md`、frontmatter、目录树和文本文件内容
- 冲突检测、问题聚合、Dashboard 统计
- 技能 attach 同步和关联关系追踪

## Requirements

- Go 1.24+
- SQLite for local development, or PostgreSQL if you want an external database

## Run Locally

```bash
cp .env.example .env
make run
```

Default server address: `http://localhost:8080`

Seed default providers on startup:

```bash
SEED=true make run
```

Seed once and exit:

```bash
make seed
```

From the repository root, you can also start the full stack with seeded data:

```bash
make dev/seed
```

## Configuration

The default development config is defined in `.env.example`.

- `PORT`: HTTP port
- `LOG_LEVEL`: log verbosity
- `DB_DRIVER`: `sqlite` or `postgres`
- `DB_DSN`: database DSN
- `SEED`: seed default providers on startup
- `SEED_ONLY`: seed and exit without serving requests

When `SEED=true`, SKM checks a small set of common local skill directories under the current user's home directory and only inserts providers for paths that actually exist.

## Core Endpoints

```text
GET  /healthz
GET  /version

GET  /api/dashboard

GET  /api/providers
POST /api/providers
GET  /api/providers/:zid
PUT  /api/providers/:zid
DELETE /api/providers/:zid
POST /api/providers/:zid/scan

POST /api/scan
GET  /api/scan-jobs
GET  /api/scan-jobs/:zid
GET  /api/issues
GET  /api/issues?view=latest
GET  /api/conflicts

GET  /api/skills
GET  /api/skills?grouped=true
GET  /api/skills/:zid
POST /api/skills/:zid/attach
GET  /api/skills/:zid/files
GET  /api/skills/:zid/file-content?path=SKILL.md
```

Detailed request and response examples live in `../docs/frontend-api-contract.md`.

## Attach Mode

`POST /api/skills/:zid/attach` request body:

```json
{
  "targetProviderZid": "PROV0001",
  "mode": "attach"
}
```

- `mode=move`: move the whole skill directory into the target Provider, then rescan both sides
- `mode=attach`: copy a filtered subset of files into the target Provider using `.to` include and exclude rules, write `.from` into the target, then rescan the target

When you call `GET /api/skills?grouped=true`, attached copies tracked by `.from` are folded under the source skill's `relatedSkills` field.

Example `.to` metadata:

```json
{
  "directories": ["/abs/path/to/copied-skill"],
  "include": ["README.md", "cmd/**", "internal/**/*.go"],
  "exclude": ["**/*_test.go", "bin/**"]
}
```

## Data Model

- `providers`
- `skills`
- `scan_jobs`
- `scan_issues`

The default SQLite database file is `./data/app.db` inside the repository, or `~/.skm/app.db` when SKM runs outside the repository working directory.

## Development

```bash
make test
make build
```

If you are working on the frontend at the same time, start from the repository root with `make dev`.

## License

MIT. See `../LICENSE`.
