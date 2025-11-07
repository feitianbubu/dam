package example

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type ProjectRouter struct{}

func (p *ProjectRouter) InitProjectRouter(Router *gin.RouterGroup) {
	projectRouter := Router.Group("project").Use(middleware.OperationRecord())
	projectRouterWithoutRecord := Router.Group("project")
	{
		// 需要记录操作的路由
		projectRouter.POST("create", exaProjectApi.CreateProject)       // 创建项目
		projectRouter.PUT("update", exaProjectApi.UpdateProject)        // 更新项目
		projectRouter.DELETE("delete", exaProjectApi.DeleteProject)     // 删除项目
		projectRouter.POST("permission/set", exaProjectApi.SetProjectPermission)       // 设置项目权限
		projectRouter.POST("permission/remove", exaProjectApi.RemoveProjectPermission) // 移除项目权限
	}
	{
		// 不需要记录操作的路由
		projectRouterWithoutRecord.GET("info", exaProjectApi.GetProjectInfo)                   // 获取项目详情
		projectRouterWithoutRecord.POST("list", exaProjectApi.GetProjectList)                  // 获取项目列表
		projectRouterWithoutRecord.GET("permission/list", exaProjectApi.GetProjectPermissions) // 获取项目权限列表
	}
}
