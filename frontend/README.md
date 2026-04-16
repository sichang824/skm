# SKM Frontend

SKM 前端提供 Skills Manager 的 Web 控制台，包括 Dashboard、技能浏览、Provider 管理、异常与冲突视图，以及与桌面宿主共用的单页应用界面。

## Features

- Dashboard 汇总卡片、最近扫描记录和最新问题列表
- 技能列表、搜索、筛选和详情弹窗
- Provider 的新增、编辑、启停、删除和手动扫描
- 异常列表与冲突分组视图
- 兼容浏览器开发模式和 Wails 桌面模式

## Requirements

- Node.js 20+
- pnpm 10+

## Quick Start

Install dependencies:

```bash
make install
```

Run frontend only:

```bash
make dev
```

Run the full stack from the repository root:

```bash
cd ..
make dev
```

Default dev URL: `http://localhost:5173`

## API Configuration

In development, Vite proxies `/api` requests to `http://localhost:8080` by default.

If you want to bypass the proxy, set:

```bash
VITE_API_BASE_URL=http://localhost:8080
```

If you start the stack from the repository root with different ports, the root Makefile will pass the proxy target for you.

## Routes

- `/` and `/dashboard`: Dashboard
- `/skills`: skills list
- `/skills/:zid`: skill detail modal route
- `/providers`: Provider management
- `/issues`: issues and conflicts

## Scripts

```bash
make install   # install dependencies
make dev       # start Vite dev server
make preview   # preview the production build
make test      # run tests
make lint      # run ESLint
make type-check
make build     # production build
make clean     # remove dist
```

Vitest, Testing Library and jsdom are used for UI tests.

## API Contract

See `../docs/frontend-api-contract.md` for endpoint mapping and example payloads.

## Notes

The frontend intentionally uses a thin data layer based on `fetch` and local component state so that API behavior stays easy to trace during development.
