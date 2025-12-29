package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
)

// VectorSearchParams 向量搜索参数
type VectorSearchParams struct {
	Prompt       string   `json:"prompt" form:"prompt"`                                          // 自然语言搜索提示词
	KnowledgeIDs []string `json:"knowledgeIds" form:"knowledgeIds" swaggerignore:"true"`         // 知识库ID列表（向量搜索时必需）
	TopK         int      `json:"topK" form:"topK" example:"10" swaggerignore:"true"`            // 向量搜索返回结果数量，默认10
	MinScore     float64  `json:"minScore" form:"minScore" example:"0.5" swaggerignore:"true"`   // 最小相似度分数，默认0.0
	SearchType   int      `json:"searchType" form:"searchType" example:"2" swaggerignore:"true"` // 检索类型：0=语义检索，1=全文检索，2=混合检索，默认2
}

// ExaFileSearchRequest 文件搜索请求，支持传统搜索和向量搜索
type ExaFileSearchRequest struct {
	Keyword  string   `json:"keyword" form:"keyword"`
	ClassId  int      `json:"classId" form:"classId"`   // 分类ID（内部统一使用此字段）
	Tags     []string `json:"tags" form:"tags"`         // 标签过滤，支持多个标签
	UserId   uint     `json:"userId" form:"userId"`     // 用户ID过滤
	Username string   `json:"username" form:"username"` // 用户名过滤（可选，用于显示）
	Etag     string   `json:"etag" form:"etag"`         // Etag过滤
	VectorSearchParams
	request.PageInfo
}

func (r *ExaFileSearchRequest) IsVectorSearch() bool {
	return len(r.Prompt) > 0
}

func (r *ExaFileSearchRequest) GetSearchDefaults() {
	if r.VectorSearchParams.TopK <= 0 {
		r.VectorSearchParams.TopK = 30
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

func (r *ExaFileSearchRequest) getDefaultKnowledgeIDs() []string {
	// 如果指定了classId，返回该分类的知识库ID
	if r.ClassId > 0 {
		var category example.ExaAttachmentCategory
		if err := global.GVA_DB.Where("id = ?", r.ClassId).First(&category).Error; err == nil && category.KnowledgeID != "" {
			return []string{category.KnowledgeID}
		}
		return []string{}
	}

	// 如果classId未填，返回所有分类的知识库ID
	var categories []example.ExaAttachmentCategory
	if err := global.GVA_DB.Where("knowledge_id != ?", "").Find(&categories).Error; err == nil {
		knowledgeIDs := make([]string, 0, len(categories))
		for _, category := range categories {
			if category.KnowledgeID != "" {
				knowledgeIDs = append(knowledgeIDs, category.KnowledgeID)
			}
		}
		return knowledgeIDs
	}

	return []string{}
}

// Validate 验证请求参数
func (r *ExaFileSearchRequest) Validate() error {
	// 向量搜索时不再强制要求提供KnowledgeIDs，允许使用默认值
	return nil
}
