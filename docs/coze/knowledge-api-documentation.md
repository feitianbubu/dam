# Coze Studio Knowledge Base API Documentation

## 概述

Coze Studio 知识库 API 提供了完整的知识库管理功能，包括知识库的创建、文档上传、处理和检索等操作。本文档详细说明了所有可用的 API 端点，帮助开发者快速集成知识库功能。

### 基础信息

- **基础 URL**: `https://coze.clinx.work`
- **开发环境**: `http://localhost:8888`
- **协议**: HTTPS/HTTP
- **认证方式**: Bearer Token 或 API Key
- **数据格式**: JSON

### 支持的功能

- ✅ 知识库 CRUD 操作
- ✅ 文档上传和管理（文本、表格、图片格式）
- ✅ 文档处理和分段
- ✅ 图片管理和描述生成
- ✅ 表格结构验证和管理
- ✅ 文档切片和分块策略

---

## 认证

所有 API 请求都需要认证。支持以下两种认证方式：

### Bearer Token 认证
```http
Authorization: Bearer YOUR_JWT_TOKEN
```

### API Key 认证
```http
X-API-Key: YOUR_API_KEY
```

---

## 错误处理

API 使用标准 HTTP 状态码。所有响应都包含以下基础字段：

```json
{
  "code": 0,        // 业务状态码，0 表示成功
  "msg": "success", // 响应消息
  "data": {}        // 响应数据
}
```

### 常见错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 认证失败 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 数据类型定义

### FormatType - 知识库格式类型

| 值 | 类型 | 说明 |
|---|------|------|
| 0 | Text | 文本文档 |
| 1 | Table | 表格数据 |
| 2 | Image | 图片 |
| 3 | Database | 数据库 |

### DocumentStatus - 文档状态

| 值 | 状态 | 说明 |
|---|------|------|
| 0 | Processing | 处理中/上传中 |
| 1 | Enable | 启用/就绪 |
| 2 | Disable | 禁用/失败 |
| 3 | Deleted | 已删除 |
| 4 | Resegment | 重新分段中 |
| 5 | Refreshing | 刷新中 |
| 9 | Failed | 失败 |

### ChunkType - 分块策略类型

| 值 | 类型 | 说明 |
|---|------|------|
| 0 | DefaultChunk | 默认分块 |
| 1 | CustomChunk | 自定义分块 |
| 2 | LevelChunk | 层级分块 |

---

## API 端点

## 1. 知识库管理

### 1.1 创建知识库

创建一个新的知识库。

**端点**: `POST /api/knowledge/create`

**请求体**:
```json
{
  "name": "产品文档库",
  "description": "完整的产品文档和用户指南",
  "space_id": "1234567890",
  "format_type": 0,
  "icon_uri": "tos://bucket/path/to/icon.png",
  "project_id": "9876543210",
  "biz_id": "0"
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "dataset_id": "1122334455"
}
```

**字段说明**:
- `name` (必填): 知识库名称，最大长度 100 字符
- `description`: 知识库描述
- `space_id` (必填): 所属空间 ID
- `format_type` (必填): 知识库格式类型
- `icon_uri`: 自定义图标 URI
- `project_id`: 项目 ID
- `biz_id`: 第三方业务标识，Coze 传 0

### 1.2 获取知识库列表

获取知识库的分页列表，支持筛选和排序。

**端点**: `POST /api/knowledge/list`

**请求体**:
```json
{
  "space_id": "1234567890",
  "page": 1,
  "size": 20,
  "filter": {
    "name": "文档",
    "format_type": 0,
    "scope_type": 1
  },
  "order_field": 2,
  "order_type": 1,
  "need_ref_bots": false
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "dataset_list": [
    {
      "dataset_id": "1122334455",
      "name": "产品文档库",
      "description": "完整的产品文档和用户指南",
      "format_type": 0,
      "status": 1,
      "doc_count": 5,
      "slice_count": 127,
      "create_time": 1704067200,
      "update_time": 1704067200,
      "creator_id": "9876543210",
      "space_id": "1234567890",
      "can_edit": true
    }
  ],
  "total": 42
}
```

### 1.3 获取知识库详情

获取指定知识库的详细信息。

**端点**: `POST /api/knowledge/detail`

**请求体**:
```json
{
  "dataset_ids": ["1122334455", "2233445566"],
  "space_id": "1234567890",
  "project_id": "9876543210"
}
```

### 1.4 更新知识库

更新知识库的元数据和配置。

**端点**: `POST /api/knowledge/update`

**请求体**:
```json
{
  "dataset_id": "1122334455",
  "name": "更新后的产品文档库",
  "description": "更新后的描述",
  "icon_uri": "tos://bucket/new/icon.png"
}
```

### 1.5 删除知识库

软删除指定的知识库。

**端点**: `POST /api/knowledge/delete`

**请求体**:
```json
{
  "dataset_id": "1122334455"
}
```

---

## 2. 文档管理

### 2.1 上传文档

向知识库上传文档。支持多种格式和上传方式。

**端点**: `POST /api/knowledge/document/create`

**请求体**:
```json
{
  "dataset_id": "1122334455",
  "format_type": 0,
  "document_bases": [
    {
      "name": "用户手册 v2.1",
      "source_info": {
        "tos_uri": "tos://bucket/documents/manual.pdf",
        "document_source": 0
      }
    }
  ],
  "chunk_strategy": {
    "separator": ".",
    "max_tokens": "1000",
    "remove_extra_spaces": true,
    "chunk_type": 0
  },
  "parsing_strategy": {
    "parsing_type": 1,
    "image_extraction": true,
    "table_extraction": true
  }
}
```

**支持的上传方式**:

1. **TOS URI 方式**:
```json
{
  "source_info": {
    "tos_uri": "tos://bucket/documents/manual.pdf",
    "document_source": 0
  }
}
```

2. **Base64 编码方式**:
```json
{
  "source_info": {
    "file_base64": "data:application/pdf;base64,JVBERi0xLjQK...",
    "file_type": "PDF",
    "document_source": 0
  }
}
```

3. **自定义内容方式**（适用于表格类型）:
```json
{
  "source_info": {
    "custom_content": "[{\"name\":\"张三\",\"age\":\"30\"},{\"name\":\"李四\",\"age\":\"25\"}]",
    "document_source": 2
  }
}
```

4. **ImageX URI 方式**:
```json
{
  "source_info": {
    "imagex_uri": "imagex://service/path/to/image.jpg",
    "document_source": 0
  }
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "document_infos": [
    {
      "document_id": "2233445566",
      "name": "用户手册 v2.1",
      "status": 0,
      "type": "pdf",
      "size": 204800,
      "create_time": 1704067200
    }
  ]
}
```

### 2.2 获取文档列表

获取知识库中的文档列表。

**端点**: `POST /api/knowledge/document/list`

**请求体**:
```json
{
  "dataset_id": "1122334455",
  "page": 1,
  "size": 20,
  "keyword": "手册"
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "document_infos": [
    {
      "document_id": "2233445566",
      "name": "用户手册 v2.1",
      "status": 1,
      "type": "pdf",
      "size": 204800,
      "char_count": 12450,
      "slice_count": 23,
      "hit_count": 89,
      "create_time": 1704067200,
      "update_time": 1704067200,
      "creator_id": "9876543210"
    }
  ],
  "total": 15
}
```

### 2.3 更新文档

更新文档的元数据和表格结构。

**端点**: `POST /api/knowledge/document/update`

**请求体**:
```json
{
  "document_id": "2233445566",
  "document_name": "更新后的文档名称",
  "table_meta": [
    {
      "id": "1",
      "column_name": "产品名称",
      "is_semantic": true,
      "sequence": "1",
      "column_type": 1
    }
  ]
}
```

### 2.4 删除文档

删除指定的文档。

**端点**: `POST /api/knowledge/document/delete`

**请求体**:
```json
{
  "document_ids": ["2233445566", "3344556677"]
}
```

### 2.5 获取文档处理进度

检查文档的处理状态和进度。

**端点**: `POST /api/knowledge/document/progress/get`

**请求体**:
```json
{
  "document_ids": ["2233445566", "3344556677"]
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "document_id": "2233445566",
      "progress": 75,
      "status": 0,
      "document_name": "用户手册 v2.1",
      "remaining_time": "120",
      "status_descript": "正在解析文档内容..."
    }
  ]
}
```

### 2.6 重新分段文档

使用新的分块和解析策略重新处理文档。

**端点**: `POST /api/knowledge/document/resegment`

**请求体**:
```json
{
  "dataset_id": "1122334455",
  "document_ids": ["2233445566"],
  "chunk_strategy": {
    "separator": "\\n\\n",
    "max_tokens": "800",
    "chunk_type": 1,
    "overlap": "100"
  },
  "parsing_strategy": {
    "parsing_type": 1,
    "image_extraction": true,
    "table_extraction": true,
    "image_ocr": true
  }
}
```

---

## 3. 图片管理

### 3.1 获取图片列表

获取图片知识库中的图片列表。

**端点**: `POST /api/knowledge/photo/list`

**请求体**:
```json
{
  "dataset_id": "1122334455",
  "page": 1,
  "size": 20,
  "filter": {
    "has_caption": true,
    "keyword": "界面",
    "status": 1
  }
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "photo_infos": [
    {
      "document_id": "2233445566",
      "name": "产品截图.png",
      "url": "https://example.com/images/screenshot.png",
      "caption": "产品仪表板显示分析概览",
      "type": "png",
      "size": 512000,
      "status": 1,
      "create_time": 1704067200
    }
  ],
  "total": 25
}
```

### 3.2 获取图片详情

获取指定图片的详细信息。

**端点**: `POST /api/knowledge/photo/detail`

**请求体**:
```json
{
  "document_ids": ["2233445566"],
  "dataset_id": "1122334455"
}
```

### 3.3 更新图片描述

更新图片的描述/标题。

**端点**: `POST /api/knowledge/photo/caption`

**请求体**:
```json
{
  "document_id": "2233445566",
  "caption": "更新后的产品界面显示新功能"
}
```

### 3.4 AI 提取图片描述

使用 AI 视觉模型自动生成图片描述。

**端点**: `POST /api/knowledge/photo/extract_caption`

**请求体**:
```json
{
  "document_id": "2233445566"
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "caption": "一个显示图表和指标的 Web 应用程序仪表板截图"
}
```

---

## 4. 表格架构管理

### 4.1 获取表格架构

获取上传表格文件的结构和预览数据。

**端点**: `POST /api/knowledge/table_schema/get`

**请求体**:
```json
{
  "source_file": {
    "tos_uri": "tos://bucket/tables/data.xlsx"
  },
  "table_sheet": {
    "sheet_id": "0",
    "header_line_idx": "0",
    "start_line_idx": "1"
  },
  "table_data_type": 0
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "sheet_list": [
    {
      "id": "0",
      "sheet_name": "Sheet1",
      "total_row": "1000"
    }
  ],
  "table_meta": [
    {
      "id": "1",
      "column_name": "产品名称",
      "is_semantic": true,
      "sequence": "1",
      "column_type": 1
    }
  ],
  "preview_data": [
    {
      "产品名称": "iPhone 15",
      "价格": "999",
      "库存": "100"
    }
  ]
}
```

### 4.2 验证表格架构

验证表格架构是否与上传的文件结构匹配。

**端点**: `POST /api/knowledge/table_schema/validate`

**请求体**:
```json
{
  "space_id": "1234567890",
  "document_id": "2233445566",
  "source_file": {
    "tos_uri": "tos://bucket/tables/data.xlsx"
  },
  "table_sheet": {
    "sheet_id": "0",
    "header_line_idx": "0",
    "start_line_idx": "1"
  }
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "column_valid_result": {
    "产品名称": "valid",
    "价格": "valid",
    "库存": "type_mismatch"
  }
}
```

---

## 5. 工具接口

### 5.1 获取默认图标

根据知识库格式类型获取默认图标。

**端点**: `POST /api/knowledge/icon/get`

**请求体**:
```json
{
  "format_type": 0
}
```

**响应**:
```json
{
  "code": 0,
  "msg": "success",
  "icon": {
    "url": "https://example.com/icons/text.png",
    "uri": "tos://bucket/icons/text.png"
  }
}
```

---

## 使用示例

### 完整的文档上传流程

以下是一个完整的文档上传和处理流程示例：

```javascript
// 1. 创建知识库
const createKnowledgeBase = async () => {
  const response = await fetch('https://coze.clinx.work/api/knowledge/create', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer YOUR_TOKEN'
    },
    body: JSON.stringify({
      name: '产品文档库',
      description: '完整的产品文档和用户指南',
      space_id: '1234567890',
      format_type: 0,
      project_id: '9876543210'
    })
  });

  const result = await response.json();
  return result.dataset_id;
};

// 2. 上传文档
const uploadDocument = async (datasetId, fileUri) => {
  const response = await fetch('https://coze.clinx.work/api/knowledge/document/create', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer YOUR_TOKEN'
    },
    body: JSON.stringify({
      dataset_id: datasetId,
      format_type: 0,
      document_bases: [{
        name: '用户手册 v2.1',
        source_info: {
          tos_uri: fileUri,
          document_source: 0
        }
      }],
      chunk_strategy: {
        separator: '.',
        max_tokens: '1000',
        remove_extra_spaces: true,
        chunk_type: 0
      },
      parsing_strategy: {
        parsing_type: 1,
        image_extraction: true,
        table_extraction: true
      }
    })
  });

  const result = await response.json();
  return result.document_infos[0].document_id;
};

// 3. 检查处理进度
const checkProgress = async (documentId) => {
  const response = await fetch('https://coze.clinx.work/api/knowledge/document/progress/get', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer YOUR_TOKEN'
    },
    body: JSON.stringify({
      document_ids: [documentId]
    })
  });

  const result = await response.json();
  return result.data[0];
};

// 使用示例
const main = async () => {
  try {
    // 创建知识库
    const datasetId = await createKnowledgeBase();
    console.log('知识库创建成功:', datasetId);

    // 上传文档
    const documentId = await uploadDocument(datasetId, 'tos://bucket/docs/manual.pdf');
    console.log('文档上传成功:', documentId);

    // 轮询检查处理进度
    let progress = await checkProgress(documentId);
    while (progress.status === 0) {
      console.log(`处理进度: ${progress.progress}%`);
      await new Promise(resolve => setTimeout(resolve, 2000));
      progress = await checkProgress(documentId);
    }

    if (progress.status === 1) {
      console.log('文档处理完成！');
    } else {
      console.error('文档处理失败:', progress.status_descript);
    }

  } catch (error) {
    console.error('操作失败:', error);
  }
};
```

### Python 使用示例

```python
import requests
import time
import json

class CozeKnowledgeAPI:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.headers = {
            'Content-Type': 'application/json',
            'Authorization': f'Bearer {token}'
        }

    def create_knowledge_base(self, name, description, space_id, format_type=0):
        """创建知识库"""
        url = f"{self.base_url}/api/knowledge/create"
        data = {
            "name": name,
            "description": description,
            "space_id": space_id,
            "format_type": format_type
        }

        response = requests.post(url, headers=self.headers, json=data)
        response.raise_for_status()
        return response.json()["dataset_id"]

    def upload_document(self, dataset_id, name, file_content_base64, file_type):
        """上传文档"""
        url = f"{self.base_url}/api/knowledge/document/create"
        data = {
            "dataset_id": dataset_id,
            "format_type": 0,
            "document_bases": [{
                "name": name,
                "source_info": {
                    "file_base64": file_content_base64,
                    "file_type": file_type,
                    "document_source": 0
                }
            }],
            "chunk_strategy": {
                "separator": ".",
                "max_tokens": "1000",
                "remove_extra_spaces": True,
                "chunk_type": 0
            }
        }

        response = requests.post(url, headers=self.headers, json=data)
        response.raise_for_status()
        return response.json()["document_infos"][0]["document_id"]

    def check_document_progress(self, document_id):
        """检查文档处理进度"""
        url = f"{self.base_url}/api/knowledge/document/progress/get"
        data = {"document_ids": [document_id]}

        response = requests.post(url, headers=self.headers, json=data)
        response.raise_for_status()
        return response.json()["data"][0]

    def list_documents(self, dataset_id, page=1, size=20):
        """获取文档列表"""
        url = f"{self.base_url}/api/knowledge/document/list"
        data = {
            "dataset_id": dataset_id,
            "page": page,
            "size": size
        }

        response = requests.post(url, headers=self.headers, json=data)
        response.raise_for_status()
        return response.json()

# 使用示例
if __name__ == "__main__":
    api = CozeKnowledgeAPI("https://coze.clinx.work", "YOUR_TOKEN")

    # 创建知识库
    dataset_id = api.create_knowledge_base(
        name="测试知识库",
        description="用于测试的知识库",
        space_id="1234567890"
    )
    print(f"知识库创建成功: {dataset_id}")

    # 上传文档（这里需要先将文件转换为 base64）
    # with open("document.pdf", "rb") as f:
    #     file_content = base64.b64encode(f.read()).decode()
    #     document_id = api.upload_document(dataset_id, "测试文档", file_content, "PDF")
    #     print(f"文档上传成功: {document_id}")
```

---

## 最佳实践

### 1. 文档上传建议

- **文件大小**: 建议单个文件不超过 50MB
- **支持格式**: PDF、DOCX、TXT、MD、HTML 等文本格式；XLSX、CSV 等表格格式；JPG、PNG 等图片格式
- **分块策略**: 根据文档类型选择合适的分块策略，文本类文档建议使用默认策略，技术文档可考虑层级分块

### 2. 性能优化

- **批量操作**: 尽量使用批量接口减少 API 调用次数
- **分页查询**: 大量数据查询时使用合适的分页大小（建议 20-50）
- **进度轮询**: 文档处理进度检查建议间隔 2-5 秒

### 3. 错误处理

- **重试机制**: 对于网络超时等临时错误实现重试机制
- **状态检查**: 上传文档后及时检查处理状态，处理失败时查看错误描述
- **参数验证**: 在客户端先验证必填参数，减少无效请求

### 4. 安全注意事项

- **Token 管理**: 妥善保管认证 Token，避免泄露
- **文件验证**: 上传前验证文件类型和大小
- **权限控制**: 确保只操作有权限的知识库和文档

---

## 更新日志

### v1.0.0 (2024-01-01)
- 初始版本发布
- 支持基础的知识库 CRUD 操作
- 支持文档上传和管理
- 支持图片管理和 AI 描述生成
- 支持表格架构验证

---

## 联系支持

如果您在使用过程中遇到问题或有建议，请通过以下方式联系我们：

- **GitHub Issues**: https://github.com/coze-dev/coze-studio/issues
- **文档地址**: https://docs.coze.com
- **API 状态页面**: https://status.coze.com

---

*本文档基于 Coze Studio 后端 IDL 文件自动生成，如有疑问请参考最新的源码实现。*