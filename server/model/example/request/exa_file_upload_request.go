package request

import "strings"

type ExaFileUploadRequest struct {
	ClassId int    `json:"classId" form:"classId"`
	Tags    string `json:"tags" form:"tags"`
	Owner   string `json:"owner" form:"owner"`
}

func (req *ExaFileUploadRequest) ParseTags() []string {
	if req.Tags == "" {
		return []string{}
	}

	// 清理标签并分割
	tags := strings.Split(req.Tags, ",")
	var parsedTags []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			parsedTags = append(parsedTags, tag)
		}
	}

	return parsedTags
}
