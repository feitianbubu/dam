package coze

// Coze API 特定的数据结构定义

// CozeCreateKnowledgeBaseRequest Coze创建知识库请求
type CozeCreateKnowledgeBaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SpaceID     string `json:"space_id"`
	FormatType  int    `json:"format_type"`
	IconURI     string `json:"icon_uri,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	BizID       string `json:"biz_id"`
}

// CozeCreateKnowledgeBaseResponse Coze创建知识库响应
type CozeCreateKnowledgeBaseResponse struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	DatasetID string `json:"dataset_id"`
}

// CozeListKnowledgeBasesRequest Coze获取知识库列表请求
type CozeListKnowledgeBasesRequest struct {
	SpaceID     string                 `json:"space_id"`
	Page        int                    `json:"page"`
	Size        int                    `json:"size"`
	Filter      map[string]interface{} `json:"filter,omitempty"`
	OrderField  int                    `json:"order_field,omitempty"`
	OrderType   int                    `json:"order_type,omitempty"`
	NeedRefBots bool                   `json:"need_ref_bots,omitempty"`
}

// CozeKnowledgeBaseInfo Coze知识库信息
type CozeKnowledgeBaseInfo struct {
	DatasetID   string `json:"dataset_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FormatType  int    `json:"format_type"`
	Status      int    `json:"status"`
	DocCount    int    `json:"doc_count"`
	SliceCount  int    `json:"slice_count"`
	CreateTime  int64  `json:"create_time"`
	UpdateTime  int64  `json:"update_time"`
	CreatorID   string `json:"creator_id"`
	SpaceID     string `json:"space_id"`
	CanEdit     bool   `json:"can_edit"`
}

// CozeListKnowledgeBasesResponse Coze获取知识库列表响应
type CozeListKnowledgeBasesResponse struct {
	Code        int                      `json:"code"`
	Msg         string                   `json:"msg"`
	DatasetList []*CozeKnowledgeBaseInfo `json:"dataset_list"`
	Total       int                      `json:"total"`
}

// CozeUploadDocumentRequest Coze上传文档请求
type CozeUploadDocumentRequest struct {
	DatasetID       string               `json:"dataset_id"`
	FormatType      int                  `json:"format_type"`
	DocumentBases   []CozeDocumentBase   `json:"document_bases"`
	ChunkStrategy   *CozeChunkStrategy   `json:"chunk_strategy,omitempty"`
	ParsingStrategy *CozeParsingStrategy `json:"parsing_strategy,omitempty"`
}

// CozeDocumentBase Coze文档基础信息
type CozeDocumentBase struct {
	Name       string         `json:"name"`
	SourceInfo CozeSourceInfo `json:"source_info"`
}

// CozeSourceInfo Coze文档源信息
type CozeSourceInfo struct {
	TosURI         string `json:"tos_uri,omitempty"`
	FileBase64     string `json:"file_base64,omitempty"`
	FileType       string `json:"file_type,omitempty"`
	CustomContent  string `json:"custom_content,omitempty"`
	ImagexURI      string `json:"imagex_uri,omitempty"`
	DocumentSource int    `json:"document_source"`
}

// CozeChunkStrategy Coze分块策略
type CozeChunkStrategy struct {
	Separator         string `json:"separator,omitempty"`
	MaxTokens         int    `json:"max_tokens,omitempty"`
	RemoveExtraSpaces bool   `json:"remove_extra_spaces,omitempty"`
	ChunkType         int    `json:"chunk_type,omitempty"`
	Overlap           int    `json:"overlap,omitempty"`
}

// CozeParsingStrategy Coze解析策略
type CozeParsingStrategy struct {
	ParsingType     int  `json:"parsing_type,omitempty"`
	ImageExtraction bool `json:"image_extraction,omitempty"`
	TableExtraction bool `json:"table_extraction,omitempty"`
	ImageOCR        bool `json:"image_ocr,omitempty"`
}

// CozeDocumentInfo Coze文档信息
type CozeDocumentInfo struct {
	DocumentID string `json:"document_id"`
	Name       string `json:"name"`
	Status     int    `json:"status"`
	Type       string `json:"type"`
	Size       int    `json:"size"`
	CharCount  int    `json:"char_count,omitempty"`
	SliceCount int    `json:"slice_count,omitempty"`
	HitCount   int    `json:"hit_count,omitempty"`
	CreateTime int64  `json:"create_time"`
	UpdateTime int64  `json:"update_time,omitempty"`
	CreatorID  string `json:"creator_id,omitempty"`
}

// CozeUploadDocumentResponse Coze上传文档响应
type CozeUploadDocumentResponse struct {
	Code          int                 `json:"code"`
	Msg           string              `json:"msg"`
	DocumentInfos []*CozeDocumentInfo `json:"document_infos"`
}

// CozeListDocumentsRequest Coze获取文档列表请求
type CozeListDocumentsRequest struct {
	DatasetID string `json:"dataset_id"`
	Page      int    `json:"page"`
	Size      int    `json:"size"`
	Keyword   string `json:"keyword,omitempty"`
}

// CozeListDocumentsResponse Coze获取文档列表响应
type CozeListDocumentsResponse struct {
	Code          int                 `json:"code"`
	Msg           string              `json:"msg"`
	DocumentInfos []*CozeDocumentInfo `json:"document_infos"`
	Total         int                 `json:"total"`
}

// CozeDocumentProgressRequest Coze获取文档处理进度请求
type CozeDocumentProgressRequest struct {
	DocumentIDs []string `json:"document_ids"`
}

// CozeDocumentProgressInfo Coze文档处理进度信息
type CozeDocumentProgressInfo struct {
	DocumentID     string `json:"document_id"`
	Progress       int    `json:"progress"`
	Status         int    `json:"status"`
	DocumentName   string `json:"document_name"`
	RemainingTime  string `json:"remaining_time"`
	StatusDescript string `json:"status_descript"`
}

// CozeDocumentProgressResponse Coze获取文档处理进度响应
type CozeDocumentProgressResponse struct {
	Code int                         `json:"code"`
	Msg  string                      `json:"msg"`
	Data []*CozeDocumentProgressInfo `json:"data"`
}

// CozeDeleteDocumentRequest Coze删除文档请求
type CozeDeleteDocumentRequest struct {
	DocumentIDs []string `json:"document_ids"`
}

// CozeDeleteKnowledgeBaseRequest Coze删除知识库请求
type CozeDeleteKnowledgeBaseRequest struct {
	DatasetID string `json:"dataset_id"`
}

// CozeBaseResponse Coze基础响应
type CozeBaseResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Coze状态码映射
const (
	// Coze文档状态 -> 通用状态
	CozeDocumentStatusProcessing = 0 // Processing
	CozeDocumentStatusEnable     = 1 // Enable
	CozeDocumentStatusDisable    = 2 // Disable
	CozeDocumentStatusDeleted    = 3 // Deleted
	CozeDocumentStatusResegment  = 4 // Resegment
	CozeDocumentStatusRefreshing = 5 // Refreshing
	CozeDocumentStatusFailed     = 9 // Failed
)

// Coze文档源类型
const (
	CozeDocumentSourceFile   = 0 // 文件上传
	CozeDocumentSourceURL    = 1 // URL链接
	CozeDocumentSourceCustom = 2 // 自定义内容
)

// CozeKnowledgeRetrieveRequest Coze知识库检索请求
type CozeKnowledgeRetrieveRequest struct {
	Query              string                `json:"query"`                          // 查询文本
	KnowledgeIDs       []string              `json:"knowledge_ids"`                  // 知识库ID列表
	TopK               int                   `json:"top_k,omitempty"`                // 返回结果数量，默认10
	MinScore           float64               `json:"min_score,omitempty"`            // 最小相似度分数，默认0.0
	SearchType         int                   `json:"search_type,omitempty"`          // 检索类型：0=语义检索，1=全文检索，2=混合检索，默认2
	EnableQueryRewrite bool                  `json:"enable_query_rewrite,omitempty"` // 是否启用查询重写，默认true
	EnableRerank       bool                  `json:"enable_rerank,omitempty"`        // 是否启用重排序，默认true
	EnableNL2SQL       bool                  `json:"enable_nl2sql,omitempty"`        // 是否启用NL2SQL，默认false
	Strategy           *CozeRetrieveStrategy `json:"strategy,omitempty"`             // 高级策略配置
}

// CozeRetrieveStrategy Coze检索策略
type CozeRetrieveStrategy struct {
	SearchType         int     `json:"search_type"`
	TopK               int     `json:"top_k"`
	MinScore           float64 `json:"min_score"`
	EnableQueryRewrite bool    `json:"enable_query_rewrite"`
	EnableRerank       bool    `json:"enable_rerank"`
	EnableNL2SQL       bool    `json:"enable_nl2sql"`
	IsPersonalOnly     bool    `json:"is_personal_only"`
}

// CozeRetrieveResult Coze检索结果
type CozeRetrieveResult struct {
	SliceID      int64                  `json:"slice_id"`      // 片段ID
	DocumentID   int64                  `json:"document_id"`   // 文档ID
	DocumentName string                 `json:"document_name"` // 文档名称
	KnowledgeID  int64                  `json:"knowledge_id"`  // 知识库ID
	Score        float64                `json:"score"`         // 相似度分数
	Content      string                 `json:"content"`       // 匹配的内容片段
	ContentType  string                 `json:"content_type"`  // 内容类型
	DocumentURL  string                 `json:"document_url"`  // 文档URL
	CreatedAt    int64                  `json:"created_at"`    // 创建时间
	UpdatedAt    int64                  `json:"updated_at"`    // 更新时间
	Extra        map[string]interface{} `json:"extra"`         // 额外信息
}

// CozeKnowledgeRetrieveResponse Coze知识库检索响应
type CozeKnowledgeRetrieveResponse struct {
	Code        int                   `json:"code"`         // 状态码
	Msg         string                `json:"msg"`          // 消息
	Results     []*CozeRetrieveResult `json:"results"`      // 检索结果
	Total       int                   `json:"total"`        // 总数
	Duration    int                   `json:"duration"`     // 耗时(毫秒)
	ActualQuery string                `json:"actual_query"` // 实际查询文本（重写后）
}
