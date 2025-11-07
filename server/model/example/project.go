package example

import "github.com/flipped-aurora/gin-vue-admin/server/global"

// Project 项目模型 - 用于文件权限管理
type Project struct {
	global.GVA_MODEL
	Name        string `json:"name" gorm:"column:name;not null;comment:项目名称" binding:"required"`
	Description string `json:"description" gorm:"column:description;comment:项目描述"`
	OwnerID     uint   `json:"ownerId" gorm:"column:owner_id;not null;comment:项目所有者ID"`

	// 可见性控制
	// true: 公开项目，GET请求支持匿名访问
	// false: 私有项目，所有操作需要JWT认证+Casbin权限验证
	IsPublic bool `json:"isPublic" gorm:"column:is_public;default:false;comment:是否公开(true=公开可读,false=需要权限)"`
}

func (Project) TableName() string {
	return "projects"
}
