# 知识库检索API文档

## 概述

知识库检索API提供了强大的知识检索能力，支持语义检索、全文检索和混合检索三种模式，适用于智能问答、文档检索等场景。

## 接口信息

- **接口路径**: `POST /api/knowledge/retrieve`
- **Content-Type**: `application/json`
- **认证方式**: 需要有效的会话认证

## 请求参数

### 基本参数

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| query | string | 是 | 查询文本，不能为空 |
| knowledge_ids | array[string] | 是 | 知识库ID列表，至少包含一个有效ID |
| top_k | integer | 否 | 返回结果数量，默认10，最大100 |
| min_score | number | 否 | 最小相似度分数，默认0.0，范围0-1 |
| search_type | integer | 否 | 检索类型：0=语义检索，1=全文检索，2=混合检索，默认2 |
| enable_query_rewrite | boolean | 否 | 是否启用查询重写，默认true |
| enable_rerank | boolean | 否 | 是否启用重排序，默认true |
| enable_nl2sql | boolean | 否 | 是否启用NL2SQL，默认false |

### 高级策略配置（可选）

当需要更精细的控制时，可以使用 `strategy` 参数：

```json
{
  "strategy": {
    "search_type": 2,
    "top_k": 10,
    "min_score": 0.5,
    "enable_query_rewrite": true,
    "enable_rerank": true,
    "enable_nl2sql": false,
    "is_personal_only": false
  }
}
```

## 响应格式

### 成功响应

```json
{
  "results": [
    {
      "slice_id": 123456,
      "document_id": 789,
      "document_name": "产品使用手册",
      "knowledge_id": 1,
      "knowledge_name": "产品知识库",
      "score": 0.85,
      "content": "这是匹配的文档内容片段...",
      "content_type": "text",
      "document_url": "https://example.com/document.pdf",
      "extra": {
        "knowledge_name": "产品知识库",
        "document_url": "https://example.com/document.pdf"
      },
      "created_at": 1640995200000,
      "updated_at": 1640995200000
    }
  ],
  "total": 1,
  "duration": 156,
  "actual_query": "重写后的查询文本"
}
```

## 请求示例

### 基础查询

```bash
curl -X POST "http://localhost:8888/api/knowledge/retrieve" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "query": "如何使用产品功能？",
    "knowledge_ids": ["1", "2"],
    "top_k": 5,
    "min_score": 0.3
  }'
```

### 混合检索查询

```bash
curl -X POST "http://localhost:8888/api/knowledge/retrieve" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "query": "产品价格和规格",
    "knowledge_ids": ["1"],
    "search_type": 2,
    "strategy": {
      "search_type": 2,
      "top_k": 10,
      "min_score": 0.5,
      "enable_query_rewrite": true,
      "enable_rerank": true,
      "enable_nl2sql": false
    }
  }'
```

## 检索类型说明

### 1. 语义检索 (search_type: 0)
- 基于向量相似度的语义匹配
- 适合概念性查询和意图理解

### 2. 全文检索 (search_type: 1)
- 基于关键词的精确匹配
- 适合专有名词和精确术语查询

### 3. 混合检索 (search_type: 2)
- 结合语义检索和全文检索的优势
- 通过重排序算法融合多种检索结果
- 提供最佳的检索质量和覆盖度

## 使用限制

- 单次查询最大返回100个结果
- 查询文本长度限制为2000字符
- 支持的知识库数量：单次查询最多10个
- 请求频率限制：每分钟最多100次请求