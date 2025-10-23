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

// OidcLogoutResponse OIDC登出响应
type OidcLogoutResponse struct {
	Enabled  bool   `json:"enabled"`  // 是否启用OIDC登出
	LogoutURL string `json:"logoutUrl"` // 登出URL
}

// OidcTokenResponse OIDC Token端点响应
type OidcTokenResponse struct {
	AccessToken  string `json:"access_token"`            // 访问令牌
	TokenType    string `json:"token_type"`              // 令牌类型
	ExpiresIn    int64  `json:"expires_in"`              // 过期时间（秒）
	RefreshToken string `json:"refresh_token,omitempty"` // 刷新令牌（可选）
	IDToken      string `json:"id_token,omitempty"`      // ID令牌（可选）
	Scope        string `json:"scope,omitempty"`         // 作用域（可选）
}