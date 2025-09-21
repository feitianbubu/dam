package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// VectorSearchParams 向量搜索参数
type VectorSearchParams struct {
	Prompt       string   `json:"prompt" form:"prompt"`                                  // 自然语言搜索提示词
	KnowledgeIDs []string `json:"knowledgeIds" form:"knowledgeIds" swaggerignore:"true"` // 知识库ID列表（向量搜索时必需）
	TopK         int      `json:"topK" form:"topK" example:"0"`                          // 向量搜索返回结果数量，默认10
	MinScore     float64  `json:"minScore" form:"minScore" example:"0"`                  // 最小相似度分数，默认0.0
	SearchType   int      `json:"searchType" form:"searchType" example:"2"`              // 检索类型：0=语义检索，1=全文检索，2=混合检索，默认2
}

// ExaFileSearchRequest 文件搜索请求，支持传统搜索和向量搜索
type ExaFileSearchRequest struct {
	ClassId int    `json:"classId" form:"classId" swaggerignore:"true"` // 分类ID
	Keyword string `json:"keyword" form:"keyword"`                      // 传统关键词搜索
	VectorSearchParams
	request.PageInfo
}

// IsVectorSearch 判断是否为向量搜索
func (r *ExaFileSearchRequest) IsVectorSearch() bool {
	return len(r.Prompt) > 0
}

// GetSearchDefaults 获取搜索默认值
func (r *ExaFileSearchRequest) GetSearchDefaults() {
	if r.VectorSearchParams.TopK <= 0 {
		r.VectorSearchParams.TopK = 10
	}
	if r.VectorSearchParams.MinScore < 0 {
		r.VectorSearchParams.MinScore = 0.0
	}
	if r.VectorSearchParams.SearchType < 0 || r.VectorSearchParams.SearchType > 2 {
		r.VectorSearchParams.SearchType = 2 // 默认混合检索
	}
	// 如果KnowledgeIDs为空，使用默认配置
	if len(r.VectorSearchParams.KnowledgeIDs) == 0 {
		r.VectorSearchParams.KnowledgeIDs = r.getDefaultKnowledgeIDs()
	}
}

// getDefaultKnowledgeIDs 获取默认知识库ID列表
func (r *ExaFileSearchRequest) getDefaultKnowledgeIDs() []string {
	// 从配置文件获取默认dataset_id作为知识库ID
	if defaultDatasetID, ok := global.GVA_CONFIG.Vectorization.Settings["default_dataset_id"].(string); ok && defaultDatasetID != "" {
		return []string{defaultDatasetID}
	}
	return []string{}
}

// Validate 验证请求参数
func (r *ExaFileSearchRequest) Validate() error {
	// 向量搜索时不再强制要求提供KnowledgeIDs，允许使用默认值
	return nil
}
