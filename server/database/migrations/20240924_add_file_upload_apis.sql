-- 云一Dam系统 - API接口版本管理
-- 迁移版本: 2024-09-24
-- 描述: 添加文件上传与下载相关的API接口
-- 兼容: MySQL, PostgreSQL, SQLite, SQL Server

-- 文件上传与下载 API接口添加
-- 获取上传文件详情 (GET)
-- MySQL/MariaDB版本 (推荐)
INSERT INTO sys_apis (path, description, api_group, method)
SELECT '/fileUploadAndDownload/getFileDetail', '获取上传文件详情', '文件上传与下载', 'GET'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_apis
    WHERE path = '/fileUploadAndDownload/getFileDetail'
    AND method = 'GET'
    AND deleted_at IS NULL
);

-- 重试文件处理 (POST)
-- MySQL/MariaDB版本 (推荐)
INSERT INTO sys_apis (path, description, api_group, method)
SELECT '/fileUploadAndDownload/retryProcessing', '重试文件处理', '文件上传与下载', 'POST'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_apis
    WHERE path = '/fileUploadAndDownload/retryProcessing'
    AND method = 'POST'
    AND deleted_at IS NULL
);

-- 验证插入结果
-- SELECT path, method, description, api_group
-- FROM sys_apis
-- WHERE path IN ('/fileUploadAndDownload/getFileDetail', '/fileUploadAndDownload/retryProcessing')
-- AND deleted_at IS NULL;