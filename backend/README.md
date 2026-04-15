# SKM Backend

SKM 后端负责 Provider 管理、本地 skill 扫描、索引存储、异常识别、冲突检测，以及为前端提供统一 REST API。

## 当前能力

- Provider 增删改查、启用停用
- 单 Provider 扫描与全量扫描
- Skill 列表、搜索、筛选、排序
- Skill 详情、`SKILL.md` 内容、frontmatter、目录树、文本文件预览
- 扫描历史、latest 问题视图、冲突分组、Dashboard 汇总

## 运行

```bash
cp .env.example .env
make run
```

默认启动地址：`http://localhost:8080`

如需首次自动写入默认 Provider，可设置：

```bash
SEED=true make run
```

如需只执行一次 seed 并立即退出，可直接运行：

```bash
make seed
```

如果从仓库根目录联调并希望启动时自动 seed，可运行：

```bash
make dev/seed
```

当前内置的默认 Provider 会按当前用户 home 目录检查这些路径，目录存在时才写入：

- `~/.workbuddy/skills`
- `~/Workspace/skills`
- `~/.agents/skills`
- `~/.cursor/skills`
- `~/.codex/skills`

## 主要接口

```bash
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
GET  /api/skills/:zid
GET  /api/skills/:zid/files
GET  /api/skills/:zid/file-content?path=SKILL.md
```

## 数据模型

- `providers`
- `skills`
- `scan_jobs`
- `scan_issues`

数据库默认使用 SQLite，文件位于 `./data/app.db`。

## 开发

```bash
make test
make build
```

前端联调 contract 与示例请求见 `../docs/frontend-api-contract.md`。

欢迎提交 Issue 和 Pull Request！

## License

MIT
