package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/gorm"
)

type SysOidcUser struct {
	global.GVA_MODEL
	Username     string `gorm:"column:username;type:varchar(64);uniqueIndex" json:"username"`           // 用户名
	Email        string `gorm:"column:email;type:varchar(100)" json:"email"`                           // 邮箱
	Nickname     string `gorm:"column:nickname;type:varchar(64)" json:"nickname"`                      // 昵称
	Avatar       string `gorm:"column:avatar;type:varchar(255)" json:"avatar"`                         // 头像
	Provider     string `gorm:"column:provider;type:varchar(64);not null" json:"provider"`             // 提供商
	Subject      string `gorm:"column:subject;type:varchar(255);not null" json:"subject"`              // 主题
	UserID       uint   `gorm:"column:user_id;type:bigint;not null" json:"userId"`                     // 关联用户ID
	SysUser      SysUser `gorm:"foreignKey:UserID" json:"sysUser"`                                     // 关联用户
}

func (SysOidcUser) TableName() string {
	return "sys_oidc_users"
}

func (o *SysOidcUser) BeforeCreate(tx *gorm.DB) error {
	return o.BeforeSave(tx)
}

func (o *SysOidcUser) BeforeSave(tx *gorm.DB) error {
	return nil
}