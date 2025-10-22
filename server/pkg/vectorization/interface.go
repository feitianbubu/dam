package vectorization

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
)

// VectorizationService 向量化服务通用接口
type VectorizationService interface {
	// 知识库管理
	CreateKnowledgeBase(req *CreateKnowledgeBaseRequest) (*KnowledgeBase, error)
	GetKnowledgeBase(id string) (*KnowledgeBase, error)
	ListKnowledgeBases(req *ListKnowledgeBasesRequest) (*KnowledgeBaseList, error)
	DeleteKnowledgeBase(id string) error

	// 文档管理
	UploadDocument(req *UploadDocumentRequest) (*Document, error)
	GetDocument(id string) (*Document, error)
	ListDocuments(knowledgeBaseID string, req *ListDocumentsRequest) (*DocumentList, error)
	DeleteDocument(id string) error

	// 状态管理
	GetDocumentProgress(id string) (*DocumentProgress, error)

	// 检索功能（为未来的混合检索预留）
	SearchDocuments(req *SearchRequest) (*SearchResult, error)
}

// 通用数据结构定义

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	SpaceID     string            `json:"space_id,omitempty"`
	FormatType  int               `json:"format_type"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// KnowledgeBase 知识库信息
type KnowledgeBase struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	DocCount    int               `json:"doc_count"`
	SliceCount  int               `json:"slice_count,omitempty"`
	CreateTime  int64             `json:"create_time"`
	UpdateTime  int64             `json:"update_time,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ListKnowledgeBasesRequest 获取知识库列表请求
type ListKnowledgeBasesRequest struct {
	Page     int                    `json:"page"`
	Size     int                    `json:"size"`
	Keyword  string                 `json:"keyword,omitempty"`
	Filter   map[string]interface{} `json:"filter,omitempty"`
	OrderBy  string                 `json:"order_by,omitempty"`
	OrderDir string                 `json:"order_dir,omitempty"`
}

// KnowledgeBaseList 知识库列表
type KnowledgeBaseList struct {
	Items []*KnowledgeBase `json:"items"`
	Total int              `json:"total"`
}

// UploadDocumentRequest 上传文档请求
type UploadDocumentRequest struct {
	KnowledgeBaseID string               `json:"knowledge_base_id"`
	Name            string               `json:"name"`
	Content         string               `json:"content"`
	ContentType     string               `json:"content_type"`
	Metadata        example.FileMetadata `json:"metadata,omitempty"`
	ChunkStrategy   *ChunkStrategy       `json:"chunk_strategy,omitempty"`
	ParsingStrategy *ParsingStrategy     `json:"parsing_strategy,omitempty"`
}

// Document 文档信息
type Document struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Status      string               `json:"status"`
	ContentType string               `json:"content_type"`
	Type        string               `json:"type,omitempty"`
	Size        int64                `json:"size"`
	CharCount   int                  `json:"char_count,omitempty"`
	SliceCount  int                  `json:"slice_count,omitempty"`
	HitCount    int                  `json:"hit_count,omitempty"`
	CreateTime  int64                `json:"create_time"`
	UpdateTime  int64                `json:"update_time,omitempty"`
	CreatorID   string               `json:"creator_id,omitempty"`
	Metadata    example.FileMetadata `json:"metadata,omitempty"`
}

// ListDocumentsRequest 获取文档列表请求
type ListDocumentsRequest struct {
	Page    int    `json:"page"`
	Size    int    `json:"size"`
	Keyword string `json:"keyword,omitempty"`
}

// DocumentList 文档列表
type DocumentList struct {
	Items []*Document `json:"items"`
	Total int         `json:"total"`
}

// DocumentProgress 文档处理进度
type DocumentProgress struct {
	DocumentID     string `json:"document_id"`
	Progress       int    `json:"progress"`
	Status         string `json:"status"`
	DocumentName   string `json:"document_name,omitempty"`
	StatusDescript string `json:"status_descript,omitempty"`
	RemainingTime  string `json:"remaining_time,omitempty"`
}

// ChunkStrategy 分块策略
type ChunkStrategy struct {
	Separator         string `json:"separator,omitempty"`
	MaxTokens         int    `json:"max_tokens,omitempty"`
	RemoveExtraSpaces bool   `json:"remove_extra_spaces,omitempty"`
	ChunkType         int    `json:"chunk_type,omitempty"`
	Overlap           int    `json:"overlap,omitempty"`
}

// ParsingStrategy 解析策略
type ParsingStrategy struct {
	ParsingType     int  `json:"parsing_type,omitempty"`
	ImageExtraction bool `json:"image_extraction,omitempty"`
	TableExtraction bool `json:"table_extraction,omitempty"`
	ImageOCR        bool `json:"image_ocr,omitempty"`
}

// SearchRequest 检索请求
type SearchRequest struct {
	Query           string            `json:"query"`
	KnowledgeBaseID string            `json:"knowledge_base_id"`
	TopK            int               `json:"top_k,omitempty"`
	Filter          map[string]string `json:"filter,omitempty"`
}

// SearchResult 检索结果
type SearchResult struct {
	Results []*SearchResultItem `json:"results"`
	Total   int                 `json:"total"`
}

// SearchResultItem 单个检索结果
type SearchResultItem struct {
	DocumentID   string            `json:"document_id"`
	DocumentName string            `json:"document_name"`
	Content      string            `json:"content"`
	Score        float64           `json:"score"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// 常用常量定义

// Provider 向量化服务提供商
type Provider string

const (
	ProviderCoze     Provider = "coze"
	ProviderOpenAI   Provider = "openai"
	ProviderPinecone Provider = "pinecone"
	ProviderWeaviate Provider = "weaviate"
)

// FormatType 知识库格式类型
const (
	FormatTypeText     = 0 // 文本文档
	FormatTypeTable    = 1 // 表格数据
	FormatTypeImage    = 2 // 图片
	FormatTypeDatabase = 3 // 数据库
)

// DocumentStatus 文档状态
const (
	DocumentStatusProcessing = "processing" // 处理中/上传中
	DocumentStatusEnable     = "enable"     // 启用/就绪
	DocumentStatusDisable    = "disable"    // 禁用/失败
	DocumentStatusDeleted    = "deleted"    // 已删除
	DocumentStatusResegment  = "resegment"  // 重新分段中
	DocumentStatusRefreshing = "refreshing" // 刷新中
	DocumentStatusFailed     = "failed"     // 失败
)

// ChunkType 分块策略类型
const (
	ChunkTypeDefault = 0 // 默认分块
	ChunkTypeCustom  = 1 // 自定义分块
	ChunkTypeLevel   = 2 // 层级分块
)
