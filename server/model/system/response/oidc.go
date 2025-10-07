package response

import "github.com/flipped-aurora/gin-vue-admin/server/model/system"

// OidcLoginResponse OIDC登录响应
type OidcLoginResponse struct {
	AuthURL string `json:"authUrl"` // 授权URL
	State   string `json:"state"`   // 状态参数
}

// OidcUserInfo OIDC用户信息
type OidcUserInfo struct {
	Subject  string `json:"sub"`    // 主题
	Username string `json:"username"` // 用户名
	Email    string `json:"email"`  // 邮箱
	Name     string `json:"name"`   // 姓名
	Nickname string `json:"nickname"` // 昵称
	Picture  string `json:"picture"` // 头像
	Provider string `json:"provider"` // 提供商
}

// OidcUserResponse OIDC用户响应
type OidcUserResponse struct {
	system.SysOidcUser
	Avatar string `json:"avatar"`
}