# 快速开始指南

这个指南将帮助你在 5 分钟内启动并运行这个后端服务器。

## 前置要求

- Go 1.24+ 
- Make (可选，但推荐)

## 步骤 1: 克隆或复制项目

```bash
# 如果你还没有这个项目
cd your-workspace
```

## 步骤 2: 安装依赖

```bash
go mod download
```

## 步骤 3: 配置环境变量

```bash
cp .env.example .env
```

默认配置使用 SQLite，无需额外设置即可运行。

## 步骤 4: 创建第一个用户

```bash
# 构建用户管理工具
make usermgr

# 创建用户
./bin/usermgr create
```

按提示输入：
- Email: `admin@example.com`
- Name: `Admin User`
- Password: `admin123`（开发环境）

## 步骤 5: 启动服务器

```bash
make run
```

或者：

```bash
go run ./cmd/server
```

服务器将在 `http://localhost:8080` 启动。

## 步骤 6: 测试 API

### 1. 检查健康状态

```bash
curl http://localhost:8080/healthz
```

应该返回: `ok`

### 2. 登录获取 Token

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }'
```

你会得到一个 JWT token，类似：

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "zid": "USERxxxxxxxxxxxx",
    "email": "admin@example.com",
    "name": "Admin User"
  }
}
```

### 3. 使用 Token 访问受保护的接口

```bash
# 替换 YOUR_TOKEN 为上一步获取的 token
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 获取当前用户信息
curl http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer $TOKEN"

# 创建一个 Item
curl -X POST http://localhost:8080/api/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Item",
    "description": "This is a test item"
  }'

# 获取 Items 列表
curl http://localhost:8080/api/items \
  -H "Authorization: Bearer $TOKEN"
```

## 下一步

### 添加你自己的模型

1. 在 `internal/models/` 创建新模型文件
2. 在 `internal/models/entities.go` 注册模型
3. 创建对应的 Repository、Service 和 Handler
4. 在 `cmd/server/main.go` 注册路由

### 切换到 PostgreSQL

编辑 `.env` 文件：

```bash
DB_DRIVER=postgres
DB_DSN=postgres://user:password@localhost:5432/dbname?sslmode=disable
```

### 查看更多示例

查看 `API_EXAMPLES.md` 了解所有 API 接口的详细使用方法。

## 常见问题

### 数据库文件在哪里？

默认在 `./data/app.db`（SQLite）

### 如何重置数据库？

```bash
rm -rf ./data/app.db
# 重启服务器会自动创建新数据库
```

### 如何修改端口？

编辑 `.env` 文件中的 `PORT` 变量，或者：

```bash
PORT=3000 make run
```

### 如何查看日志？

日志会输出到控制台。你可以修改 `.env` 中的 `LOG_LEVEL` 和 `LOG_FORMAT`：

```bash
LOG_LEVEL=debug    # debug, info, warn, error
LOG_FORMAT=json    # console, json
```

## 开发工具

### 用户管理

```bash
./bin/usermgr create   # 创建用户
./bin/usermgr list     # 列出所有用户
./bin/usermgr reset    # 重置密码
./bin/usermgr delete   # 删除用户
```

### 运行测试

```bash
make test
```

### 代码检查

```bash
make lint
```

### 构建生产版本

```bash
make build
./bin/backend-go
```

## 需要帮助？

查看 `README.md` 了解完整文档。
