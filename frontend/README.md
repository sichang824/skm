# SKM Frontend

SKM 前端已经按 demo v1 的结构接通完整控制台：Dashboard、技能列表、技能详情弹窗、Provider 管理、异常与冲突页。

## 当前能力

- 左侧导航、顶部栏、全量扫描入口与通知提示
- Dashboard：总览指标、最近扫描日志、Provider 分布、latest issues
- Skills：表格列表、全局搜索、筛选、详情弹窗
- Providers：新增、编辑、启停、删除、单独扫描
- Issues：latest issues 列表、冲突组列表

## 🚀 Quick Start

```bash
pnpm install

# or use the Makefile wrapper
make install

make dev        # Start dev server
```

Visit `http://localhost:5173` to see the app.

开发环境默认通过 Vite 代理访问后端 `http://localhost:8080`。如果不走代理，可设置：

```bash
VITE_API_BASE_URL=http://localhost:8080
```

This project uses `pnpm` as the package manager.

## 路由

- `/` 或 `/dashboard`: Dashboard
- `/skills`: 技能列表页
- `/skills/:zid`: 技能详情弹窗路由
- `/providers`: Provider 页
- `/issues`: 异常与冲突页

## 联调文档

完整 API contract 和示例请求见 `../docs/frontend-api-contract.md`。

## 常用命令

```bash
# Development
make dev              # Start dev server
make preview          # Preview production build

# Testing
make test             # Run tests once
make test-watch       # Run tests in watch mode
make test-ui          # Run tests with UI

# Code Quality
make lint             # Check code with ESLint
make lint-fix         # Fix ESLint issues
make format           # Format code with Prettier
make type-check       # TypeScript type checking

# Build
make build            # Production build
make build-analyze    # Build with bundle analyzer

# Cleanup
make clean            # Remove dist folder
make clean-all        # Remove node_modules and dist
```

## 🧪 Testing

```bash
make test           # Run all tests
make test-watch     # Watch mode
make test-ui        # Interactive UI
```

Tests are configured with Vitest + Testing Library + jsdom.

## 构建产物

After building, open `dist/stats.html` to visualize your bundle:

```bash
make build-analyze
```

## 说明

这套前端当前不引入额外数据层，直接使用 `fetch` + 本地组件状态联调后端接口。
