# SKM

[English](README.md)

SKM 是一个面向本地技能目录的 Skills Manager，用来扫描多个 Provider、索引 `SKILL.md` 元数据、识别冲突与异常，并通过 Web 控制台和 macOS 桌面应用进行可视化管理。

项目当前由三部分组成：

- Go 后端：负责扫描、索引、冲突检测和 REST API
- React 前端：负责 Dashboard、技能浏览、Provider 管理和问题视图
- Wails 桌面宿主：将前后端打包为 macOS 桌面应用

## Features

- 管理多个本地 Skill Provider，支持启用、停用、编辑和删除
- 扫描技能目录并提取 `SKILL.md`、frontmatter、目录树和文本文件内容
- 展示扫描历史、最新异常、冲突分组和 Dashboard 汇总指标
- 支持把技能内容按规则 attach 到其他 Provider，并追踪 `.to` / `.from` 元数据
- 同时支持浏览器开发模式和 macOS 桌面应用模式

## Design

SKM 的设计目标不是做一个通用文件浏览器，而是做一个围绕技能目录工作流的管理台。当前设计重点放在三件事上：

- 让本地分散的 skill provider 可以被统一发现、比较和维护
- 让 `SKILL.md`、目录结构、扫描结果和冲突信息出现在同一套界面里
- 让 Web 和桌面端共享同一套核心能力，减少双端分叉维护成本

在产品体验上，项目优先考虑这些原则：

- Local-first：默认围绕本地目录、SQLite 和可直接运行的开发体验构建
- Inspectable：扫描结果、问题视图和文件内容应当可追踪、可解释
- Incremental：先把 Provider、扫描、冲突和挂载能力做扎实，再扩展更重的协作能力
- Reusable：前端、后端和桌面宿主尽量复用同一套数据结构和接口约定

## Architecture

```text
skm/
├── backend/    # Go API service and scanner
├── frontend/   # React + Vite web UI
├── build/      # Desktop build output
├── main.go     # Wails desktop entry
├── wails.json  # Wails config
└── README.md
```

## Tech Stack

- Backend: Go 1.24, Gin, GORM, SQLite or PostgreSQL
- Frontend: React 19, TypeScript, Vite, Tailwind CSS v4, React Router v7, Vitest
- Desktop: Wails v2.12.0

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 20+
- pnpm 10+

### Run the full stack

```bash
make dev
```

默认地址：

- Backend: `http://localhost:8080`
- Frontend: `http://localhost:5173`

如需改端口：

```bash
BACKEND_PORT=18080 FRONTEND_PORT=4173 make dev
```

### 使用 seed 启动

```bash
make dev/seed
```

这个命令会重置本地 SQLite 数据库，启动前后端，并基于当前机器上常见的技能目录写入默认 Provider 记录。

### 分别启动各部分

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

### 桌面版开发模式

```bash
make app-dev
```

### 构建 macOS app

```bash
make app-build
```

生成的 `.app` 会输出到 `build/bin/`。

如果桌面版在仓库目录之外启动，SQLite 默认会落到 `~/Library/Application Support/SKM/app.db`。需要时可以通过 `DB_DSN` 覆盖。

## Common Commands

```bash
make install       # 安装前端依赖并预加载 Go modules
make dev           # 同时启动后端和前端
make dev/seed      # 重置数据库并以 seeded providers 启动
make reset         # 删除本地 SQLite 数据库文件
make seed          # 一次性写入默认 providers 后退出
make test          # 运行前后端测试
make build         # 构建后端二进制和前端产物
make app-dev       # 启动 Wails 桌面开发模式
make app-build     # 构建 macOS 桌面应用
```

## API Overview

公共接口：

```text
GET  /healthz
GET  /version
GET  /api/dashboard
```

Provider 和扫描接口：

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

Skill 接口：

```text
GET  /api/skills
GET  /api/skills?grouped=true
GET  /api/skills/:zid
POST /api/skills/:zid/attach
GET  /api/skills/:zid/files
GET  /api/skills/:zid/file-content?path=SKILL.md
```

请求和响应示例见 `docs/frontend-api-contract.md`。

## Configuration

主要运行配置位于 `backend/.env`：

- `PORT`: 后端端口，默认 `8080`
- `DB_DRIVER`: `sqlite` 或 `postgres`
- `DB_DSN`: 数据库连接串，默认 `./data/app.db`
- `SEED`: 启动时写入默认 providers
- `SEED_ONLY`: 只 seed 后退出

通过 Wails 开发模式运行时，桌面宿主也会读取 `backend/.env`。

## Repository Guide

- Backend setup and API notes: `backend/README.md`
- Frontend setup and scripts: `frontend/README.md`
- Frontend API contract: `docs/frontend-api-contract.md`
- Product notes: `docs/PRD.md`

## Contributing

欢迎提交 issue 和 pull request。

- Bug report: 尽量附上复现步骤、预期行为、实际行为，以及相关日志或截图
- Feature proposal: 先描述用户问题，再描述提议的交互或 API 设计
- Pull request: 保持范围聚焦，行为有变化时同步更新文档，并写清验证方式

完整贡献说明见 `CONTRIBUTING.md`。

## Acknowledgements

SKM 站在很多优秀开源项目之上，这里特别感谢：

- Wails: 提供桌面宿主和前后端整合能力
- Gin and GORM: 提供 Go 侧 HTTP 服务与数据访问基础设施
- React and Vite: 提供现代前端应用与开发体验
- Tailwind CSS, Radix UI, and shadcn/ui: 提供界面构建基础
- Lucide, GSAP, Motion, and Zustand: 支撑图标、动画和交互状态管理

同时，这个项目也受到本地优先工具链和技能目录工作流的启发，包括围绕 `SKILL.md` 组织能力描述、用目录作为分发单元、以及把冲突检测和可视化管理做成一等能力。

## License

MIT. See `LICENSE`.
