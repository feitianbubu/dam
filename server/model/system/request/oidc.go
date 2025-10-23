package request

// OidcLoginRequest OIDC登录请求
type OidcLoginRequest struct {
	Provider string `json:"provider" binding:"required"`
}

// OidcCallbackRequest OIDC回调请求
type OidcCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// OidcTokenRequest OIDC Token端点请求
type OidcTokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type" binding:"required"`       // 授权类型，固定为"authorization_code"
	Code         string `json:"code" form:"code"`                                      // 授权码（grant_type=authorization_code时必需）
	RedirectURI  string `json:"redirect_uri" form:"redirect_uri"`                      // 重定向URI
	ClientID     string `json:"client_id" form:"client_id"`                            // 客户端ID
	ClientSecret string `json:"client_secret" form:"client_secret"`                    // 客户端密钥
	RefreshToken string `json:"refresh_token" form:"refresh_token"`                    // 刷新令牌（grant_type=refresh_token时必需）
}