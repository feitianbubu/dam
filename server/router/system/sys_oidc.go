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
		oidcRouter.POST("auth", oidcApiInstance.GetOidcAuthURL)
		oidcRouter.POST("callback", oidcApiInstance.OidcCallback)
		oidcRouter.POST("token", oidcApiInstance.Token) // 标准OIDC Token端点
		oidcRouterWithAuth.GET("users", oidcApiInstance.GetOidcUsers)
		oidcRouterWithAuth.DELETE("unlink/:provider", oidcApiInstance.UnlinkOidc)
		oidcRouter.GET("logout-url", oidcApiInstance.GetOidcLogoutURL)
	}
}