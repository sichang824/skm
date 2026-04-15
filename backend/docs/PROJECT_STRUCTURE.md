# 项目结构说明

这个文档详细说明了项目的完整结构和每个文件的作用。

## 📁 项目文件树

```
backend-go/
├── 📄 README.md                          # 项目主文档
├── 📄 QUICKSTART.md                      # 快速开始指南
├── 📄 DEVELOPMENT.md                     # 开发指南
├── 📄 API_EXAMPLES.md                    # API 使用示例
├── 📄 LICENSE                            # MIT 许可证
├── 📄 .gitignore                         # Git 忽略文件
├── 📄 .env.example                       # 环境变量示例
├── 📄 go.mod                             # Go 模块依赖
├── 📄 Makefile                           # 构建和运行命令
├── 📄 Dockerfile                         # Docker 镜像构建
├── 📄 docker-compose.yml                 # Docker Compose 配置
│
├── 📁 cmd/                               # 应用程序入口
│   ├── 📁 server/                        # 主服务器
│   │   └── 📄 main.go                    # 服务器入口点
│   └── 📁 usermgr/                       # 用户管理 CLI
│       └── 📄 main.go                    # CLI 入口点
│
├── 📁 internal/                          # 私有应用代码
│   ├── 📁 config/                        # 配置管理
│   │   └── 📄 config.go                  # 配置加载和解析
│   │
│   ├── 📁 models/                        # 数据模型
│   │   ├── 📄 base.go                    # 基础模型（BaseModel）
│   │   ├── 📄 entities.go                # 实体注册和 ZID 管理
│   │   ├── 📄 user.go                    # 用户模型
│   │   └── 📄 item.go                    # Item 模型（示例）
│   │
│   ├── 📁 repository/                    # 数据访问层
│   │   ├── 📄 user_repo.go               # 用户数据访问
│   │   └── 📄 item_repo.go               # Item 数据访问
│   │
│   ├── 📁 service/                       # 业务逻辑层
│   │   ├── 📄 auth_service.go            # 认证服务
│   │   └── 📄 item_service.go            # Item 业务逻辑
│   │
│   ├── 📁 http/                          # HTTP 相关
│   │   ├── 📁 auth/                      # 认证相关
│   │   │   ├── 📄 jwt.go                 # JWT 令牌管理
│   │   │   ├── 📄 password.go            # 密码哈希和验证
│   │   │   └── 📄 token.go               # Token 提取工具
│   │   │
│   │   ├── 📁 handlers/                  # HTTP 处理器
│   │   │   ├── 📄 health.go              # 健康检查和版本
│   │   │   ├── 📄 auth.go                # 认证接口
│   │   │   └── 📄 items.go               # Items CRUD 接口
│   │   │
│   │   ├── 📁 middleware/                # 中间件
│   │   │   ├── 📄 cors.go                # CORS 跨域
│   │   │   ├── 📄 requestid.go           # 请求 ID
│   │   │   ├── 📄 logger.go              # 请求日志
│   │   │   └── 📄 auth.go                # JWT 认证
│   │   │
│   │   ├── 📁 response/                  # 响应工具
│   │   │   └── 📄 response.go            # 统一响应格式
│   │   │
│   │   ├── 📁 pagination/                # 分页工具
│   │   │   └── 📄 pagination.go          # 分页参数和结果
│   │   │
│   │   └── 📁 requestctx/                # 请求上下文
│   │       └── 📄 keys.go                # 上下文键定义
│   │
│   └── 📁 platform/                      # 平台层
│       ├── 📁 db/                        # 数据库
│       │   └── 📄 db.go                  # 数据库连接和迁移
│       └── 📁 log/                       # 日志
│           └── 📄 log.go                 # 日志初始化
│
├── 📁 _vendor/                           # 本地依赖
│   └── 📁 zid/                           # ZID 加密 ID 库
│       └── 📁 go/                        # Go 实现
│           └── 📁 idcodec/               # ID 编解码器
│
├── 📁 scripts/                           # 脚本工具
│   └── 📄 verify.sh                      # 项目验证脚本
│
└── 📁 data/                              # 数据目录（运行时创建）
    └── 📄 app.db                         # SQLite 数据库文件
```

## 📋 文件说明

### 根目录文件

| 文件 | 说明 |
|------|------|
| `README.md` | 项目主文档，包含特性、安装、使用说明 |
| `QUICKSTART.md` | 5 分钟快速开始指南 |
| `DEVELOPMENT.md` | 详细的开发和扩展指南 |
| `API_EXAMPLES.md` | 所有 API 接口的使用示例 |
| `LICENSE` | MIT 开源许可证 |
| `.gitignore` | Git 版本控制忽略规则 |
| `.env.example` | 环境变量配置示例 |
| `go.mod` | Go 模块依赖声明 |
| `Makefile` | 构建、运行、测试命令 |
| `Dockerfile` | Docker 镜像构建配置 |
| `docker-compose.yml` | Docker Compose 多容器配置 |

### 核心代码文件

#### 入口点 (cmd/)

- **`cmd/server/main.go`** - 主服务器入口
  - 初始化配置、日志、数据库
  - 设置路由和中间件
  - 启动 HTTP 服务器

- **`cmd/usermgr/main.go`** - 用户管理 CLI
  - 创建用户
  - 重置密码
  - 列出/删除用户

#### 配置层 (internal/config/)

- **`config.go`** - 配置管理
  - 从环境变量加载配置
  - 提供默认值
  - 支持 .env 文件

#### 模型层 (internal/models/)

- **`base.go`** - 基础模型
  - BaseModel 定义（ID、Zid、时间戳）
  - GORM 钩子（自动生成 ZID）

- **`entities.go`** - 实体注册
  - 实体元数据定义
  - ZID 编解码函数
  - 自动迁移列表

- **`user.go`** - 用户模型
  - 用户表结构
  - 邮箱、姓名、密码字段

- **`item.go`** - Item 模型（示例）
  - 演示如何定义业务模型
  - 包含所有者、标题、描述、状态

#### 数据访问层 (internal/repository/)

- **`user_repo.go`** - 用户数据访问
  - 按邮箱/ZID 查找
  - 创建、列出、删除

- **`item_repo.go`** - Item 数据访问
  - 分页列表
  - CRUD 操作

#### 业务逻辑层 (internal/service/)

- **`auth_service.go`** - 认证服务
  - 用户登录
  - JWT 生成
  - 用户信息查询

- **`item_service.go`** - Item 业务逻辑
  - 权限检查
  - 业务规则验证
  - CRUD 操作

#### HTTP 层 (internal/http/)

**认证 (auth/)**
- `jwt.go` - JWT 令牌生成和验证
- `password.go` - bcrypt 密码哈希
- `token.go` - 从请求提取 token

**处理器 (handlers/)**
- `health.go` - 健康检查和版本接口
- `auth.go` - 登录和用户信息接口
- `items.go` - Items CRUD 接口

**中间件 (middleware/)**
- `cors.go` - 跨域资源共享
- `requestid.go` - 请求追踪 ID
- `logger.go` - 请求日志记录
- `auth.go` - JWT 认证验证

**工具 (response/, pagination/, requestctx/)**
- `response.go` - 统一 API 响应格式
- `pagination.go` - 分页参数解析和结果封装
- `keys.go` - 请求上下文键定义

#### 平台层 (internal/platform/)

- **`db/db.go`** - 数据库管理
  - 支持 SQLite 和 PostgreSQL
  - 自动迁移
  - 连接池配置

- **`log/log.go`** - 日志系统
  - 基于 zap
  - 支持 console 和 json 格式
  - 可配置日志级别

## 🔑 关键特性实现

### 1. ZID 系统

**位置**: `internal/models/entities.go`

ZID 是一个加密的 ID 系统，提供：
- 类型安全（带前缀，如 USER、ITEM）
- 防枚举（加密的数字 ID）
- 短小易读

**使用**:
```go
// 自动生成（通过 GORM 钩子）
user := &User{Email: "test@example.com"}
db.Create(user)
// user.Zid 自动生成，如 "USERa1b2c3d4e5f6"

// 手动编解码
zid, _ := models.Encode("ITEM", 123)
prefix, id, _ := models.Decode(zid)
```

### 2. 分层架构

**数据流**:
```
HTTP Request
    ↓
Handler (验证请求)
    ↓
Service (业务逻辑)
    ↓
Repository (数据访问)
    ↓
Database
```

### 3. JWT 认证

**位置**: `internal/http/auth/jwt.go` + `internal/http/middleware/auth.go`

- 登录时生成 JWT
- 中间件验证 JWT
- 从 token 提取用户信息

### 4. 分页支持

**位置**: `internal/http/pagination/pagination.go`

- 自动解析 `page` 和 `pageSize` 参数
- 返回统一的分页结果格式
- 支持排序

## 🚀 扩展指南

### 添加新模型

1. 在 `internal/models/` 创建模型文件
2. 在 `entities.go` 注册模型
3. 创建对应的 repository、service、handler
4. 在 `main.go` 注册路由

详见 `DEVELOPMENT.md`

### 添加新中间件

1. 在 `internal/http/middleware/` 创建文件
2. 实现 `gin.HandlerFunc`
3. 在 `main.go` 中使用 `r.Use()`

### 切换数据库

修改 `.env`:
```bash
# SQLite
DB_DRIVER=sqlite
DB_DSN=./data/app.db

# PostgreSQL
DB_DRIVER=postgres
DB_DSN=postgres://user:pass@localhost:5432/db?sslmode=disable
```

## 📚 相关文档

- [README.md](README.md) - 项目概述和基本使用
- [QUICKSTART.md](QUICKSTART.md) - 快速开始
- [DEVELOPMENT.md](DEVELOPMENT.md) - 开发指南
- [API_EXAMPLES.md](API_EXAMPLES.md) - API 示例

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！
