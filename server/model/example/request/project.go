package request

import "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

// ProjectCreateRequest 创建项目请求
type ProjectCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsPublic    bool   `json:"isPublic"`
}

// ProjectUpdateRequest 更新项目请求
type ProjectUpdateRequest struct {
	ID          uint   `json:"id" binding:"required"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    *bool  `json:"isPublic"` // 使用指针以区分未传值和false
}

// ProjectListRequest 项目列表请求
type ProjectListRequest struct {
	request.PageInfo
	Name     string `json:"name" form:"name"`       // 项目名称（模糊搜索）
	IsPublic *bool  `json:"isPublic" form:"isPublic"` // 是否公开（可选筛选）
}

// ProjectPermissionRequest 项目权限设置请求
type ProjectPermissionRequest struct {
	ProjectID   string `json:"projectId" binding:"required"`
	AuthorityID uint   `json:"authorityId" binding:"required"` // 角色ID
	Permission  string `json:"permission" binding:"required,oneof=read write"` // read 或 write
}
