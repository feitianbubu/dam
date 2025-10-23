package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
)

// GetOidcAuthURL
// @Tags     Oidc
// @Summary  获取OIDC授权URL
// @Produce   application/json
// @Param    data  body      systemReq.OidcLoginRequest                                     true  "provider"
// @Success  200   {object}  response.Response{data=response.OidcLoginResponse,msg=string}  "返回授权URL和状态"
// @Router   /oidc/auth [post]
func (o *OidcApi) GetOidcAuthURL(c *gin.Context) {
	var req systemReq.OidcLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	authResp, err := oidcService.GetAuthURL(req.Provider)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithData(authResp, c)
}

// OidcCallback
// @Tags     Oidc
// @Summary  OIDC回调处理
// @Produce   application/json
// @Param    data  body      systemReq.OidcCallbackRequest                              true  "code, state"
// @Success  200   {object}  response.Response{data=response.LoginResponse,msg=string}  "返回用户信息和token"
// @Router   /oidc/callback [post]
func (o *OidcApi) OidcCallback(c *gin.Context) {
	var req systemReq.OidcCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	user, err := oidcService.HandleCallback(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if user.Enable != 1 {
		response.FailWithMessage("用户被禁止登录", c)
		return
	}

	new(BaseApi).TokenNext(c, *user)
}

// GetOidcUsers
// @Tags     Oidc
// @Summary  获取用户OIDC信息
// @Produce   application/json
// @Success  200   {object}  response.Response{data=[]system.SysOidcUser,msg=string}  "返回OIDC绑定列表"
// @Router   /oidc/users [get]
// @Security ApiKeyAuth
func (o *OidcApi) GetOidcUsers(c *gin.Context) {
	claims := utils.GetUserInfo(c)

	oidcUsers, err := oidcService.GetOidcUsers(claims.BaseClaims.ID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithData(oidcUsers, c)
}

// UnlinkOidc
// @Tags     !Oidc
// @Summary  解绑OIDC
// @Produce   application/json
// @Param    provider  path      string  true  "provider"
// @Success  200   {object}  response.Response{msg=string}  "解绑成功"
// @Router   /oidc/unlink/{provider} [delete]
// @Security ApiKeyAuth
func (o *OidcApi) UnlinkOidc(c *gin.Context) {
	claims := utils.GetUserInfo(c)
	provider := c.Param("provider")

	if provider == "" {
		response.FailWithMessage("provider不能为空", c)
		return
	}

	err := oidcService.UnlinkOidc(claims.BaseClaims.ID, provider)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("解绑成功", c)
}

// GetOidcLogoutURL
// @Tags     Oidc
// @Summary  获取OIDC登出URL
// @Produce   application/json
// @Param    post_logout_redirect_uri  query      string  false  "登出后重定向URI"
// @Success  200   {object}  response.Response{data=response.OidcLogoutResponse,msg=string}  "返回OIDC登出URL和状态"
// @Router   /oidc/logout-url [get]
func (o *OidcApi) GetOidcLogoutURL(c *gin.Context) {
	postLogoutRedirectURI := c.Query("post_logout_redirect_uri")

	logoutURL, err := oidcService.GetLogoutURL(postLogoutRedirectURI)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithData(logoutURL, c)
}

// Token
// @Tags     Oidc
// @Summary  OIDC Token端点
// @Accept   application/x-www-form-urlencoded
// @Accept   application/json
// @Produce  application/json
// @Param    code           formData  string  false  "授权码 (grant_type=authorization_code时必需)"
// @Param    redirect_uri   formData  string  false  "重定向URI (grant_type=authorization_code时必需)"
// @Param    client_id      formData  string  true   "客户端ID"
// @Param    client_secret  formData  string  true   "客户端密钥"
// @Param    refresh_token  formData  string  false  "刷新令牌 (grant_type=refresh_token时必需)"
// @Success  200  {object}  response.OidcTokenResponse  "返回访问令牌和相关信息"
// @Failure  400  {object}  response.Response           "请求参数错误"
// @Failure  401  {object}  response.Response           "客户端认证失败"
// @Failure  500  {object}  response.Response           "服务器内部错误"
// @Router   /oidc/token [post]
func (o *OidcApi) Token(c *gin.Context) {
	var req systemReq.OidcTokenRequest

	// 支持form-urlencoded和JSON两种格式
	contentType := c.GetHeader("Content-Type")
	if contentType == "application/x-www-form-urlencoded" || contentType == "application/json" {
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(400, gin.H{
				"error":             "invalid_request",
				"error_description": err.Error(),
			})
			return
		}
	} else {
		c.JSON(400, gin.H{
			"error":             "invalid_request",
			"error_description": "Content-Type must be application/x-www-form-urlencoded or application/json",
		})
		return
	}

	tokenResp, err := oidcService.Token(req)
	if err != nil {
		// 根据错误类型返回不同的错误响应
		if err.Error() == "无效的client_id" || err.Error() == "无效的client_secret" {
			c.JSON(401, gin.H{
				"error":             "invalid_client",
				"error_description": err.Error(),
			})
		} else if err.Error() == "OIDC功能未启用" {
			c.JSON(503, gin.H{
				"error":             "temporarily_unavailable",
				"error_description": err.Error(),
			})
		} else {
			c.JSON(400, gin.H{
				"error":             "invalid_request",
				"error_description": err.Error(),
			})
		}
		return
	}

	// 返回标准OIDC Token响应
	c.JSON(200, tokenResp)
}
