# API Examples

这个文档提供了所有 API 接口的使用示例。

## 基础信息

- **Base URL**: `http://localhost:8080`
- **认证方式**: Bearer Token (JWT)

## 1. 健康检查

### 健康状态

```bash
curl http://localhost:8080/healthz
```

**响应**:
```
ok
```

### 版本信息

```bash
curl http://localhost:8080/version
```

**响应**:
```json
{
  "version": "1.0.0",
  "status": "running"
}
```

## 2. 认证

### 登录

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "your-password"
  }'
```

**响应**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "zid": "USERxxxxxxxxxxxx",
    "email": "user@example.com",
    "name": "User Name"
  }
}
```

### 获取当前用户信息

```bash
curl http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应**:
```json
{
  "zid": "USERxxxxxxxxxxxx",
  "email": "user@example.com",
  "name": "User Name"
}
```

## 3. Items (示例资源)

### 获取列表（分页）

```bash
# 默认分页
curl http://localhost:8080/api/items \
  -H "Authorization: Bearer YOUR_TOKEN"

# 自定义分页
curl "http://localhost:8080/api/items?page=1&pageSize=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应**:
```json
{
  "items": [
    {
      "zid": "ITEMxxxxxxxxxxxx",
      "ownerZid": "USERxxxxxxxxxxxx",
      "title": "Sample Item",
      "description": "This is a sample item",
      "status": "active",
      "createdAt": "2024-01-14T10:00:00Z",
      "updatedAt": "2024-01-14T10:00:00Z"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

### 创建 Item

```bash
curl -X POST http://localhost:8080/api/items \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My New Item",
    "description": "This is a new item"
  }'
```

**响应**:
```json
{
  "zid": "ITEMxxxxxxxxxxxx",
  "ownerZid": "USERxxxxxxxxxxxx",
  "title": "My New Item",
  "description": "This is a new item",
  "status": "active",
  "createdAt": "2024-01-14T10:00:00Z",
  "updatedAt": "2024-01-14T10:00:00Z"
}
```

### 获取单个 Item

```bash
curl http://localhost:8080/api/items/ITEMxxxxxxxxxxxx \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应**:
```json
{
  "zid": "ITEMxxxxxxxxxxxx",
  "ownerZid": "USERxxxxxxxxxxxx",
  "title": "My Item",
  "description": "Item description",
  "status": "active",
  "createdAt": "2024-01-14T10:00:00Z",
  "updatedAt": "2024-01-14T10:00:00Z"
}
```

### 更新 Item

```bash
curl -X PUT http://localhost:8080/api/items/ITEMxxxxxxxxxxxx \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated Title",
    "description": "Updated description",
    "status": "completed"
  }'
```

**响应**:
```json
{
  "zid": "ITEMxxxxxxxxxxxx",
  "ownerZid": "USERxxxxxxxxxxxx",
  "title": "Updated Title",
  "description": "Updated description",
  "status": "completed",
  "createdAt": "2024-01-14T10:00:00Z",
  "updatedAt": "2024-01-14T10:30:00Z"
}
```

### 删除 Item

```bash
curl -X DELETE http://localhost:8080/api/items/ITEMxxxxxxxxxxxx \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**响应**:
```json
{
  "message": "Item deleted successfully"
}
```

## 错误响应

所有错误响应都遵循以下格式：

```json
{
  "error": "Error message description"
}
```

### 常见错误码

- `400 Bad Request` - 请求参数错误
- `401 Unauthorized` - 未认证或 token 无效
- `403 Forbidden` - 无权限访问资源
- `404 Not Found` - 资源不存在
- `500 Internal Server Error` - 服务器内部错误

## 使用 Postman

你可以导入以下环境变量到 Postman：

```json
{
  "base_url": "http://localhost:8080",
  "token": "YOUR_JWT_TOKEN_HERE"
}
```

然后在请求中使用：
- URL: `{{base_url}}/api/items`
- Authorization: Bearer Token `{{token}}`
