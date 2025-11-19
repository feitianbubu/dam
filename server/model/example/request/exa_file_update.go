package request

import (
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
)

type ExaFileUpdateRequest struct {
	ID uint `json:"id" form:"id" binding:"required" example:"1"`

	Name      *string `json:"name" form:"name" example:"updated_file.jpg"`
	ClassId   *int    `json:"classId" form:"classId" example:"1"`
	ProjectID *string `json:"projectId" form:"projectId" example:"2"`

	Tags *string `json:"tags" form:"tags" example:"cinx,test"`

	Reprocess    *bool `json:"reprocess" form:"reprocess" example:"false"`
	UpdateVector *bool `json:"updateVector" form:"updateVector" example:"false"`
	PublicRead   *bool `json:"publicRead" form:"publicRead" example:"false"`
}

// GetBizMetadata 获取业务元数据
func (r *ExaFileUpdateRequest) GetBizMetadata() *example.BizMetadata {
	bizMetadata := &example.BizMetadata{}

	if r.ProjectID != nil {
		bizMetadata.ProjectID = *r.ProjectID
	}

	if r.Tags != nil {
		bizMetadata.Tags = strings.Split(*r.Tags, ",")
	}

	// 如果没有任何字段被设置，返回nil
	if bizMetadata.ProjectID == "" && len(bizMetadata.Tags) == 0 {
		return nil
	}

	return bizMetadata
}

// parseTagsFromString 从逗号分隔的字符串解析标签
func parseTagsFromString(tagsStr string) []string {
	if tagsStr == "" {
		return []string{}
	}

	tags := strings.Split(tagsStr, ",")
	var parsedTags []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			parsedTags = append(parsedTags, tag)
		}
	}

	return parsedTags
}
