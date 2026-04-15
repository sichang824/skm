# SKM

SKM 是一个前后端分离的技能管理系统项目目录，当前包含一个 Go 后端服务和一个 React 前端应用。仓库已经具备基础的认证、示例资源管理、前端页面骨架与开发脚手架，适合作为继续实现技能管理业务的基础工程。

## 目录结构

```text
skm/
├── backend/   # Go + Gin + GORM 后端服务
├── frontend/  # React 19 + Vite + TypeScript 前端应用
└── README.md
```

## 技术栈

### Backend

- Go 1.24
- Gin
- GORM
- SQLite / PostgreSQL
- JWT 鉴权
- ZID（加密友好 ID）

### Frontend

- React 19
- TypeScript
- Vite
- Tailwind CSS v4
- shadcn/ui
- React Router v7
- Vitest

## 当前能力

目前项目已经提供以下基础能力：

- 后端健康检查与版本接口
- 基于 JWT 的登录与当前用户信息查询
- 一个受保护的示例资源 `items` 的增删改查接口
- 用户管理 CLI，用于创建、删除、列出和重置用户
- 前端首页、组件展示页、扩展页和 404 页面
- 前后端各自独立的 Makefile 开发命令

需要注意的是，`frontend` 当前仍然更偏向通用模板展示页，`backend` 中的 `items` 也是示例业务资源。如果要把它落成真正的 Skills Manager，后续还需要把“技能、分类、标签、熟练度、检索、后台管理”等领域模型和页面补齐。

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

### 3. 创建测试用户

```bash
cd backend
make usermgr
./bin/usermgr create
```

如果启用了种子数据，通常会自动生成示例账号，详见 `backend/README.md`。

### 4. 单独启动前端

```bash
cd frontend
make install
make dev
```

默认访问地址：`http://localhost:5173`

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
```

## 后端接口概览

### 公共接口

```text
GET  /healthz
GET  /version
POST /auth/login
```

### 需要认证的接口

```text
GET    /api/auth/me
GET    /api/items
POST   /api/items
GET    /api/items/:zid
PUT    /api/items/:zid
DELETE /api/items/:zid
```

认证方式为 Bearer Token。登录成功后，将返回的 JWT 放入请求头：

```text
Authorization: Bearer <token>
```

## 配置说明

后端主要通过 `backend/.env` 配置：

- `PORT`：服务端口，默认 `8080`
- `DB_DRIVER`：数据库驱动，默认 `sqlite`
- `DB_DSN`：数据库连接串，默认 `./data/app.db`
- `SEED`：是否初始化示例数据
- `JWT_SECRET`：JWT 签名密钥
- `JWT_TOKEN_DURATION`：Token 有效期，默认 `24h`

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
