# 云一Dam数据库版本管理

## 目录结构
```
database/
├── migrations/          # SQL迁移文件
├── README.md           # 本文件
└── example.cnf         # 数据库连接配置模板
```

## 迁移文件命名规范
- 格式: `YYYYMMDD_描述性名称.sql`
- 示例: `20240924_add_file_upload_apis.sql`

## 使用说明

### 执行迁移
```bash
# MySQL
mysql -u username -p database_name < migrations/20240924_add_file_upload_apis.sql

# PostgreSQL  (注意使用PostgreSQL兼容语法)
psql -U username -d database_name -f migrations/20240924_add_file_upload_apis.sql

# SQLite
sqlite3 database.db < migrations/20240924_add_file_upload_apis.sql
```

### 多数据库兼容性
每份迁移文件都包含多个版本的SQL语句：
- **MySQL**: 使用 `INSERT ... ON DUPLICATE KEY UPDATE`
- **PostgreSQL**: 使用 `DO $$` 块处理异常
- **SQLite**: 使用 `INSERT OR IGNORE`

使用对应数据库的语法块，注释掉其他版本。

## 迁移历史

| 日期 | 文件 | 描述 | 状态 |
|------|------|------|------|
| 2024-09-24 | 20240924_add_file_upload_apis.sql | 添加文件上传下载相关API | ✅ |

## 注意事项
1. 迁移前先备份数据库
2. 执行前检查目标数据库类型
3. 确认使用对应数据库的SQL语法版本
4. 每个文件都包含验证SQL，执行后可验证结果