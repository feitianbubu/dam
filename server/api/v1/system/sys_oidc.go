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
// @Success  200   {object}  response.Response{data=systemRes.OidcLoginResponse,msg=string}  "返回授权URL和状态"
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
// @Success  200   {object}  response.Response{data=systemRes.LoginResponse,msg=string}  "返回用户信息和token"
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

	// 检查用户状态
	if user.Enable != 1 {
		response.FailWithMessage("用户被禁止登录", c)
		return
	}

	// 签发JWT token
	new(BaseApi).TokenNext(c, *user)
}

// GetOidcUsers
// @Tags     Oidc
// @Summary  获取用户的OIDC绑定
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
// @Tags     Oidc
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
// @Success  200   {object}  response.Response{data=map[string]interface{},msg=string}  "返回登出URL"
// @Router   /oidc/logout-url [get]
func (o *OidcApi) GetOidcLogoutURL(c *gin.Context) {
	logoutURL, err := oidcService.GetLogoutURL("")
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithData(gin.H{
		"logoutUrl": logoutURL,
		"enabled":   logoutURL != "",
	}, c)
}