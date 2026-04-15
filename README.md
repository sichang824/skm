# SKM

SKM 是一个前后端分离的技能管理系统项目目录，当前包含一个 Go 后端服务、一个 React 前端应用，以及一个基于 Wails 的 macOS 桌面宿主。项目当前聚焦于本地 Skill Provider 的扫描、索引、冲突识别与可视化管理。

## 目录结构

```text
skm/
├── backend/    # Go + Gin + GORM 后端服务
├── frontend/   # React 19 + Vite + TypeScript 前端应用
├── main.go     # Wails 桌面宿主入口
├── wails.json  # Wails 构建配置
└── README.md
```

## 技术栈

### Backend 命令

- Go 1.24
- Gin
- GORM
- SQLite / PostgreSQL
- JWT 鉴权
- ZID（加密友好 ID）

### Frontend 命令

- React 19
- TypeScript
- Vite
- Tailwind CSS v4
- shadcn/ui
- React Router v7
- Vitest

### Desktop

- Wails v2.12.0
- macOS `.app` 打包

## 当前能力

目前项目已经提供以下基础能力：

- Provider 的增删改查、启用停用与手动扫描
- 全量扫描、扫描历史、Dashboard 汇总
- Skill 列表、详情、目录文件浏览、`SKILL.md` 预览
- Skill 冲突分组、异常检测、latest 问题视图
- 前端控制台界面与 macOS 桌面版打包
- 前后端各自独立的 Makefile 开发命令

需要注意的是，当前项目已经不是通用模板，但仍处于持续演进阶段。核心链路已经围绕 Skills Manager 落地，后续可以继续补充更细的领域模型、批量编辑能力和更完整的桌面交互体验。

## 快速开始

### 1. 一键启动前后端

```bash
make dev
```

默认后端地址为 `http://localhost:8080`，默认前端地址为 `http://localhost:5173`。

如果需要改端口，可直接覆盖：

```bash
BACKEND_PORT=18080 FRONTEND_PORT=4173 make dev
```

如需带 seed 启动联调：

```bash
make dev/seed
```

该命令会先删除本地 SQLite 数据库文件，再以全新数据库启动前后端并执行默认 Provider seed。

如需只写入默认 Provider 而不启动长驻服务：

```bash
make seed
```

如需只重置本地数据库：

```bash
make reset
```

### 2. 单独启动后端

```bash
cd backend
cp .env.example .env
make run
```

默认监听地址：`http://localhost:8080`

如果希望首次启动时写入示例数据，可将 `.env` 中的 `SEED=false` 改为 `SEED=true`。

### 3. 执行一次 seed 或扫描

```bash
make seed
```

如需启动后立刻做一次全量扫描，可进入前端后点击“全局扫描”，或直接调用后端扫描接口。

### 4. 单独启动前端

```bash
cd frontend
make install
make dev
```

默认访问地址：`http://localhost:5173`

### 5. 启动桌面版

```bash
make app-dev
```

该命令会按 `wails.json` 的配置启动 Wails 宿主，并复用现有 `frontend/` 目录作为前端工程。

桌面版在 macOS 上使用隐藏标题栏模式：

- 原生标题文字隐藏，仅保留左上角红黄绿按钮
- 顶部应用工具栏仍可用于拖动窗口
- 页面内按钮区域保持正常点击，不会被拖拽行为吞掉

### 6. 构建 macOS app

```bash
make app-build
```

构建完成后，macOS `.app` 会输出到 `build/bin/` 目录。

### 7. 桌面版数据目录

桌面版会根据运行环境自动选择 SQLite 路径：

- 仓库内开发模式默认使用 `backend/data/app.db`
- 打包后的 `.app` 默认使用 `~/Library/Application Support/SKM/app.db`

这样 Finder 双击打开 `.app` 时，不会因为工作目录不同而出现数据库文件找不到的问题。

### 8. 给桌面版执行 seed

如果要给打包后的桌面版数据库初始化默认 Provider 数据，可以执行：

```bash
SEED=true SEED_ONLY=true DB_DSN="$HOME/Library/Application Support/SKM/app.db" /Users/ann/Workspace/skills/skm/build/bin/skm.app/Contents/MacOS/skm
```

如果你要初始化的是仓库开发环境数据库，则继续使用：

```bash
make seed
```

两者的差别是：

- `make seed` 作用于仓库内开发数据库
- 上面的 `.app` 命令作用于桌面版实际使用的数据库

## 常用命令

### Backend

```bash
cd backend

make run      # 启动服务
make test     # 运行测试
make build    # 构建服务端二进制
make usermgr  # 构建用户管理 CLI
make clean    # 清理构建产物
```

### Frontend

```bash
cd frontend

make install     # 安装依赖
make dev         # 启动开发服务器
make test        # 运行测试
make lint        # 运行 ESLint
make type-check  # TypeScript 类型检查
make build       # 生产构建
```

### Workspace

```bash
make dev          # 同时启动后端和前端
make dev/seed     # 先重置本地数据库，再启动前后端，并在后端启动时执行 Provider seed
make reset        # 删除本地 SQLite 数据库文件
make seed         # 一次性写入默认 Provider
make dev-backend  # 仅启动后端
make dev-frontend # 仅启动前端
make test         # 运行前后端测试
make build        # 构建前后端
make app-dev      # 启动 Wails 桌面开发模式
make app-build    # 构建 macOS 桌面 app
```

## 后端接口概览

### 公共接口

```text
GET  /healthz
GET  /version
GET  /api/dashboard
```

### Provider 与扫描接口

```text
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
```

### Skill 接口

```text
GET  /api/skills
GET  /api/skills?grouped=true
GET  /api/skills/:zid
POST /api/skills/:zid/attach
GET  /api/skills/:zid/files
GET  /api/skills/:zid/file-content?path=SKILL.md
```

## 配置说明

后端主要通过 `backend/.env` 配置：

- `PORT`：服务端口，默认 `8080`
- `DB_DRIVER`：数据库驱动，默认 `sqlite`
- `DB_DSN`：数据库连接串，默认 `./data/app.db`
- `SEED`：是否初始化示例数据

通过 Wails 从仓库根目录启动时，也会自动读取 `backend/.env`，SQLite 默认路径会优先落到 `backend/data/app.db`。

如果桌面版从 Finder、Launchpad 或其他非仓库目录启动，SQLite 路径会自动解析到 `~/Library/Application Support/SKM/app.db`。如需覆盖，可显式设置 `DB_DSN`。

## 桌面版排错

如果你遇到桌面版启动后直接退出，优先检查这几项：

1. 前端资源是否已经成功构建，先执行 `cd frontend && make build`
2. `backend/.env` 是否存在，且数据库配置可用
3. 桌面版数据库目录是否可写，默认是 `~/Library/Application Support/SKM/`
4. 是否需要先执行一次 seed，避免初始数据为空

直接从命令行启动桌面版二进制，通常比 Finder 更容易看到错误：

```bash
/Users/ann/Workspace/skills/skm/build/bin/skm.app/Contents/MacOS/skm
```

前端如需接入真实后端接口，建议下一步补充统一的 API Base URL 配置，并在 `src` 中增加数据访问层。

## 开发建议

如果你准备继续把 SKM 做成真正的技能管理系统，推荐按这个顺序推进：

1. 在后端定义技能相关实体，例如 `skills`、`categories`、`tags`、`levels`。
2. 将现有 `items` 示例接口替换或扩展为实际业务接口。
3. 在前端补充 API 请求封装、登录态管理和真实列表/详情/编辑页面。
4. 为前后端增加联调配置、权限设计和基础 E2E 验证。

## 参考文档

- `backend/README.md`：后端详细说明
- `backend/docs/QUICKSTART.md`：后端快速开始
- `backend/docs/API_EXAMPLES.md`：接口示例
- `frontend/README.md`：前端详细说明

## 当前状态总结

这个目录目前更准确地说是一个“Skills Manager 的项目骨架”，而不是已经完成业务闭环的成品。它已经具备继续开发所需的基础设施，但业务模型、页面流程和前后端真实联动仍需要按你的 SKM 目标继续落地。
