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