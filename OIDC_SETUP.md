# OIDC 第三方登录配置指南

本文档介绍如何在云一Dam系统中配置和使用OIDC第三方登录功能。

## 功能概述

实现的OIDC最小功能包括：

1. **OIDC授权流程**：支持标准的Authorization Code Flow
2. **用户信息获取**：从OIDC提供商获取用户基本信息
3. **自动用户创建**：支持首次登录时自动创建用户
4. **用户绑定管理**：支持查看和解绑OIDC账号
5. **JWT Token集成**：与现有JWT认证系统集成

## 后端配置

### 1. 配置文件设置

在 `server/config.yaml` 中添加OIDC配置：

```yaml
# oidc configuration
oidc:
    enabled: true                                    # 启用OIDC功能
    provider: "google"                              # OIDC提供商名称
    client-id: "your-client-id"                     # 客户端ID
    client-secret: "your-client-secret"             # 客户端密钥
    redirect-url: "http://localhost:8888/oidc/callback"  # 回调地址
    scopes: "openid profile email"                  # 请求的权限范围
    issuer: "https://accounts.google.com"           # 发行者URL
    auth-url: "https://accounts.google.com/o/oauth2/v2/auth"  # 授权URL
    token-url: "https://oauth2.googleapis.com/token" # 令牌URL
    userinfo-url: "https://openidconnect.googleapis.com/v1/userinfo" # 用户信息URL
    end-session-url: ""                             # 注销URL（可选）
    auto-create-user: true                          # 自动创建用户
    default-authority: 888                          # 新用户默认权限ID
```

### 2. 支持的OIDC提供商示例

#### Google OAuth2
```yaml
oidc:
    enabled: true
    provider: "google"
    client-id: "your-google-client-id.apps.googleusercontent.com"
    client-secret: "your-google-client-secret"
    redirect-url: "http://localhost:8888/oidc/callback"
    scopes: "openid profile email"
    issuer: "https://accounts.google.com"
    auth-url: "https://accounts.google.com/o/oauth2/v2/auth"
    token-url: "https://oauth2.googleapis.com/token"
    userinfo-url: "https://openidconnect.googleapis.com/v1/userinfo"
```

#### GitHub OAuth
```yaml
oidc:
    enabled: true
    provider: "github"
    client-id: "your-github-client-id"
    client-secret: "your-github-client-secret"
    redirect-url: "http://localhost:8888/oidc/callback"
    scopes: "user:email"
    issuer: "https://github.com"
    auth-url: "https://github.com/login/oauth/authorize"
    token-url: "https://github.com/login/oauth/access_token"
    userinfo-url: "https://api.github.com/user"
```

#### Microsoft Azure AD
```yaml
oidc:
    enabled: true
    provider: "microsoft"
    client-id: "your-azure-client-id"
    client-secret: "your-azure-client-secret"
    redirect-url: "http://localhost:8888/oidc/callback"
    scopes: "openid profile email"
    issuer: "https://login.microsoftonline.com/{tenant-id}/v2.0"
    auth-url: "https://login.microsoftonline.com/{tenant-id}/oauth2/v2.0/authorize"
    token-url: "https://login.microsoftonline.com/{tenant-id}/oauth2/v2.0/token"
    userinfo-url: "https://graph.microsoft.com/v1.0/me"
```

### 3. 数据库表

系统会自动创建 `sys_oidc_users` 表来存储OIDC用户绑定关系：

```sql
CREATE TABLE `sys_oidc_users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  `deleted_at` datetime(3) NULL,
  `username` varchar(64) NOT NULL,
  `email` varchar(100) DEFAULT NULL,
  `nickname` varchar(64) DEFAULT NULL,
  `avatar` varchar(255) DEFAULT NULL,
  `provider` varchar(64) NOT NULL,
  `subject` varchar(255) NOT NULL,
  `user_id` bigint NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_sys_oidc_users_deleted_at` (`deleted_at`),
  UNIQUE KEY `idx_username` (`username`),
  KEY `fk_oidc_user` (`user_id`),
  CONSTRAINT `fk_oidc_user` FOREIGN KEY (`user_id`) REFERENCES `sys_users` (`id`)
);
```

## 前端配置

### 1. 登录页面集成

登录页面已集成OIDC登录组件，会显示第三方登录按钮：

```vue
<!-- OIDC第三方登录 -->
<div v-if="showOidc" class="mt-6">
  <el-divider content-position="center">
    <span class="text-gray-500 text-sm">第三方登录</span>
  </el-divider>
  <OidcLogin />
</div>
```

### 2. 组件配置

OIDC登录组件位于 `web/src/components/oidc/oidcLogin.vue`，支持：

- Google登录
- GitHub登录
- Microsoft登录

可以通过修改 `providers` 数组来添加或修改支持的提供商。

## API接口

### 1. 获取授权URL
```http
POST /oidc/auth
Content-Type: application/json

{
  "provider": "google"
}
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "authUrl": "https://accounts.google.com/o/oauth2/v2/auth?...",
    "state": "random_state_string"
  },
  "msg": "success"
}
```

### 2. OIDC回调
```http
POST /oidc/callback
Content-Type: application/json

{
  "code": "authorization_code",
  "state": "random_state_string"
}
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "user": {...},
    "token": "jwt_token",
    "expiresAt": 1234567890
  },
  "msg": "登录成功"
}
```

### 3. 获取用户OIDC绑定
```http
GET /oidc/users
Authorization: Bearer {jwt_token}
```

### 4. 解绑OIDC
```http
DELETE /oidc/unlink/{provider}
Authorization: Bearer {jwt_token}
```

## 安全注意事项

1. **客户端密钥保护**：确保客户端密钥安全存储，不要泄露到前端
2. **重定向URL验证**：确保重定向URL与在OIDC提供商注册的URL一致
3. **状态参数验证**：使用随机状态参数防止CSRF攻击
4. **用户信息验证**：验证从OIDC提供商获取的用户信息
5. **HTTPS使用**：生产环境必须使用HTTPS

## 故障排除

### 1. 授权URL获取失败
- 检查OIDC配置是否正确
- 确认enabled设置为true
- 检查客户端ID和密钥

### 2. 回调处理失败
- 检查重定向URL配置
- 确认客户端密钥正确
- 查看服务器日志获取详细错误信息

### 3. 用户创建失败
- 检查auto-create-user配置
- 确认default-authority对应的权限存在
- 检查数据库连接

### 4. Token验证失败
- 检查JWT配置
- 确认Redis连接正常
- 查看token黑名单状态

## 扩展功能

当前实现的是最小功能集，可以根据需要扩展：

1. **多提供商支持**：同时支持多个OIDC提供商
2. **用户映射配置**：自定义用户信息映射规则
3. **权限同步**：从OIDC提供商同步用户权限信息
4. **单点注销**：实现完整的单点注销功能
5. **审计日志**：记录OIDC登录操作的审计日志