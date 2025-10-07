package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type OidcRouter struct{}

func (s *OidcRouter) InitOidcRouter(Router *gin.RouterGroup) {
	oidcRouter := Router.Group("oidc")
	oidcRouterWithAuth := Router.Group("oidc").Use(middleware.JWTAuth())
	{
		oidcRouter.POST("auth", oidcApiInstance.GetOidcAuthURL)      // 获取授权URL
		oidcRouter.POST("callback", oidcApiInstance.OidcCallback)    // OIDC回调
		oidcRouter.GET("logout-url", oidcApiInstance.GetOidcLogoutURL) // 获取登出URL
		oidcRouterWithAuth.GET("users", oidcApiInstance.GetOidcUsers) // 获取OIDC绑定
		oidcRouterWithAuth.DELETE("unlink/:provider", oidcApiInstance.UnlinkOidc) // 解绑OIDC
	}
}