# 开发指南

这个文档提供了扩展和定制这个后端框架的详细指南。

## 项目架构

### 分层架构

```
┌─────────────────────────────────────┐
│         HTTP Handlers               │  ← 处理 HTTP 请求/响应
├─────────────────────────────────────┤
│         Service Layer               │  ← 业务逻辑
├─────────────────────────────────────┤
│       Repository Layer              │  ← 数据访问
├─────────────────────────────────────┤
│         Models/Entities             │  ← 数据模型
└─────────────────────────────────────┘
```

### 目录结构说明

- `cmd/` - 应用程序入口点
  - `server/` - 主服务器
  - `usermgr/` - 用户管理 CLI
- `internal/` - 私有应用代码
  - `config/` - 配置管理
  - `models/` - 数据模型定义
  - `repository/` - 数据访问层
  - `service/` - 业务逻辑层
  - `http/` - HTTP 相关
    - `handlers/` - 请求处理器
    - `middleware/` - 中间件
    - `auth/` - 认证相关
    - `response/` - 响应工具
    - `pagination/` - 分页工具
  - `platform/` - 平台层
    - `db/` - 数据库连接
    - `log/` - 日志系统
- `_vendor/zid/` - ZID 库（本地依赖）

## 添加新功能

### 1. 创建新模型

在 `internal/models/` 创建新文件，例如 `product.go`：

```go
package models

type Product struct {
    BaseModel
    OwnerZid    string  `gorm:"type:varchar(16);index;not null" json:"ownerZid"`
    Name        string  `gorm:"type:varchar(255);not null" json:"name"`
    Price       float64 `gorm:"type:decimal(10,2);not null" json:"price"`
    Description string  `gorm:"type:text" json:"description,omitempty"`
    Stock       int     `gorm:"default:0" json:"stock"`
}
```

### 2. 注册模型

在 `internal/models/entities.go` 的 `Entities` 数组中添加：

```go
var Entities = []EntityMeta{
    {Table: "products", Prefix: "PROD", Model: &Product{}, AutoMigrate: true},
    // ... 其他模型
}
```

### 3. 创建 Repository

在 `internal/repository/` 创建 `product_repo.go`：

```go
package repository

import (
    "backend-go/internal/models"
    "gorm.io/gorm"
)

type ProductRepository struct {
    db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
    return &ProductRepository{db: db}
}

func (r *ProductRepository) List(ownerZid string, offset, limit int) ([]models.Product, int64, error) {
    var products []models.Product
    var total int64
    
    query := r.db.Where("owner_zid = ?", ownerZid)
    
    if err := query.Model(&models.Product{}).Count(&total).Error; err != nil {
        return nil, 0, err
    }
    
    if err := query.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
        return nil, 0, err
    }
    
    return products, total, nil
}

func (r *ProductRepository) FindByZid(zid string) (*models.Product, error) {
    var product models.Product
    if err := r.db.Where("zid = ?", zid).First(&product).Error; err != nil {
        return nil, err
    }
    return &product, nil
}

func (r *ProductRepository) Create(product *models.Product) error {
    return r.db.Create(product).Error
}

func (r *ProductRepository) Update(product *models.Product) error {
    return r.db.Save(product).Error
}

func (r *ProductRepository) Delete(product *models.Product) error {
    return r.db.Delete(product).Error
}
```

### 4. 创建 Service

在 `internal/service/` 创建 `product_service.go`：

```go
package service

import (
    "backend-go/internal/models"
    "backend-go/internal/repository"
    "errors"
    "gorm.io/gorm"
)

var ErrProductNotFound = errors.New("product not found")

type ProductService struct {
    productRepo *repository.ProductRepository
}

func NewProductService(productRepo *repository.ProductRepository) *ProductService {
    return &ProductService{productRepo: productRepo}
}

type CreateProductRequest struct {
    Name        string  `json:"name" binding:"required"`
    Price       float64 `json:"price" binding:"required,gt=0"`
    Description string  `json:"description"`
    Stock       int     `json:"stock" binding:"gte=0"`
}

func (s *ProductService) Create(ownerZid string, req CreateProductRequest) (*models.Product, error) {
    product := &models.Product{
        OwnerZid:    ownerZid,
        Name:        req.Name,
        Price:       req.Price,
        Description: req.Description,
        Stock:       req.Stock,
    }
    
    if err := s.productRepo.Create(product); err != nil {
        return nil, err
    }
    
    return product, nil
}

// 添加其他方法...
```

### 5. 创建 Handler

在 `internal/http/handlers/` 创建 `products.go`：

```go
package handlers

import (
    "backend-go/internal/http/pagination"
    "backend-go/internal/service"
    "net/http"
    "github.com/gin-gonic/gin"
)

type ProductHandler struct {
    productService *service.ProductService
}

func NewProductHandler(productService *service.ProductService) *ProductHandler {
    return &ProductHandler{productService: productService}
}

func (h *ProductHandler) List(c *gin.Context) {
    ownerZid, _ := c.Get("ownerZid")
    params := pagination.Parse(c)
    
    products, total, err := h.productService.List(ownerZid.(string), params.Page, params.PageSize)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
        return
    }
    
    result := pagination.Result[any]{
        Items:    make([]any, len(products)),
        Page:     params.Page,
        PageSize: params.PageSize,
        Total:    total,
    }
    for i, p := range products {
        result.Items[i] = p
    }
    
    c.JSON(http.StatusOK, result)
}

func (h *ProductHandler) Create(c *gin.Context) {
    ownerZid, _ := c.Get("ownerZid")
    
    var req service.CreateProductRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    product, err := h.productService.Create(ownerZid.(string), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
        return
    }
    
    c.JSON(http.StatusCreated, product)
}

// 添加其他方法...
```

### 6. 注册路由

在 `cmd/server/main.go` 中添加路由：

```go
// 在 api 组中添加
productRepo := repository.NewProductRepository(gdb)
productService := service.NewProductService(productRepo)
productHandler := handlers.NewProductHandler(productService)

api.GET("/products", productHandler.List)
api.POST("/products", productHandler.Create)
api.GET("/products/:zid", productHandler.Get)
api.PUT("/products/:zid", productHandler.Update)
api.DELETE("/products/:zid", productHandler.Delete)
```

## 中间件开发

### 创建自定义中间件

在 `internal/http/middleware/` 创建新文件：

```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

func RateLimiter(logger *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 实现限流逻辑
        c.Next()
    }
}
```

在 `main.go` 中使用：

```go
r.Use(middleware.RateLimiter(logger))
```

## 数据库迁移

### 自动迁移

框架会在启动时自动运行迁移。新模型会自动创建表。

### 手动迁移

如果需要更复杂的迁移，可以使用 GORM 的迁移功能：

```go
// 在 db.Open 之后
db.AutoMigrate(&models.YourModel{})
```

## 测试

### 单元测试示例

```go
package service_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestProductService_Create(t *testing.T) {
    // 设置测试数据库
    // 创建 service
    // 测试创建功能
    
    assert.NotNil(t, product)
    assert.Equal(t, "Test Product", product.Name)
}
```

运行测试：

```bash
go test ./...
```

## ZID 系统

### ZID 是什么？

ZID 是一个加密的、用户友好的 ID 系统，提供：
- 加密的数字 ID（防止枚举攻击）
- 类型前缀（如 USER、ITEM、PROD）
- 短小且易读

### 使用 ZID

```go
// 编码
zid, err := models.Encode("PROD", 123)
// 结果: "PRODa1b2c3d4e5f6"

// 解码
prefix, id, err := models.Decode("PRODa1b2c3d4e5f6")
// prefix: "PROD", id: 123
```

### 配置 ZID

在 `.env` 中设置：

```bash
ZID_MASTER_KEY=your-32-byte-master-key-here
ZID_TWEAK=v1|myapp|production
```

## 日志

### 使用日志

```go
import "go.uber.org/zap"

logger.Info("message", zap.String("key", "value"))
logger.Error("error occurred", zap.Error(err))
logger.Debug("debug info", zap.Any("data", data))
```

### 日志级别

- `debug` - 详细调试信息
- `info` - 一般信息
- `warn` - 警告信息
- `error` - 错误信息

## 性能优化

### 数据库连接池

在 `internal/platform/db/db.go` 中调整：

```go
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(5)
sqlDB.SetConnMaxLifetime(30 * time.Minute)
```

### 查询优化

使用索引、预加载关联等：

```go
db.Preload("User").Find(&items)
db.Where("status = ?", "active").Find(&items)
```

## 部署

### 使用 Docker

```bash
docker build -t backend-go .
docker run -p 8080:8080 backend-go
```

### 使用 Docker Compose

```bash
docker-compose up -d
```

### 生产环境配置

1. 修改 JWT_SECRET
2. 使用 PostgreSQL
3. 设置合适的日志级别
4. 配置 CORS 白名单
5. 启用 HTTPS

## 最佳实践

1. **错误处理** - 始终检查并处理错误
2. **验证输入** - 使用 binding tags 验证请求
3. **事务** - 对多个数据库操作使用事务
4. **日志** - 记录重要操作和错误
5. **测试** - 为关键功能编写测试
6. **文档** - 保持 API 文档更新

## 常见问题

### 如何添加新的认证方式？

在 `internal/http/auth/` 添加新的认证实现，然后在中间件中使用。

### 如何实现文件上传？

使用 Gin 的文件上传功能，参考 experts-backend-go 的 documents handler。

### 如何实现 WebSocket？

使用 gorilla/websocket 库，参考 experts-backend-go 的 subscription handler。

### 如何实现缓存？

可以集成 Redis 或使用内存缓存库如 go-cache。
