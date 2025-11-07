package example

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	exampleReq "github.com/flipped-aurora/gin-vue-admin/server/model/example/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProjectApi struct{}

// CreateProject
// @Tags      Dam
// @Summary   创建项目
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      exampleReq.ProjectCreateRequest  true  "创建项目请求"
// @Success   200   {object}  response.Response{data=Project,msg=string}  "创建成功"
// @Router    /project/create [post]
func (p *ProjectApi) CreateProject(c *gin.Context) {
	var req exampleReq.ProjectCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	userID := utils.GetUserID(c)
	project, err := projectService.CreateProject(req, userID)
	if err != nil {
		global.GVA_LOG.Error("创建项目失败!", zap.Error(err))
		response.FailWithMessage("创建项目失败", c)
		return
	}

	response.OkWithDetailed(project, "创建成功", c)
}

// UpdateProject
// @Tags      Dam
// @Summary   更新项目
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      exampleReq.ProjectUpdateRequest  true  "更新项目请求"
// @Success   200   {object}  response.Response{msg=string}  "更新成功"
// @Router    /project/update [put]
func (p *ProjectApi) UpdateProject(c *gin.Context) {
	var req exampleReq.ProjectUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	userID := utils.GetUserID(c)
	err := projectService.UpdateProject(req, userID)
	if err != nil {
		global.GVA_LOG.Error("更新项目失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("更新成功", c)
}

// DeleteProject
// @Tags      Dam
// @Summary   删除项目
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     id  query      int  true  "项目ID"
// @Success   200   {object}  response.Response{msg=string}  "删除成功"
// @Router    /project/delete [delete]
func (p *ProjectApi) DeleteProject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 32)
	if err != nil {
		response.FailWithMessage("无效的项目ID", c)
		return
	}

	userID := utils.GetUserID(c)
	err = projectService.DeleteProject(uint(id), userID)
	if err != nil {
		global.GVA_LOG.Error("删除项目失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

// GetProjectInfo
// @Tags      Dam
// @Summary   获取项目详情
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     id  query      int  true  "项目ID"
// @Success   200   {object}  response.Response{data=Project,msg=string}  "获取成功"
// @Router    /project/info [get]
func (p *ProjectApi) GetProjectInfo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 32)
	if err != nil {
		response.FailWithMessage("无效的项目ID", c)
		return
	}

	project, err := projectService.GetProjectInfo(uint(id))
	if err != nil {
		global.GVA_LOG.Error("获取项目详情失败!", zap.Error(err))
		response.FailWithMessage("获取项目详情失败", c)
		return
	}

	response.OkWithDetailed(project, "获取成功", c)
}

// GetProjectList
// @Tags      Dam
// @Summary   获取项目列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      exampleReq.ProjectListRequest  true  "获取项目列表请求"
// @Success   200   {object}  response.Response{data=response.PageResult,msg=string}  "获取成功"
// @Router    /project/list [post]
func (p *ProjectApi) GetProjectList(c *gin.Context) {
	var req exampleReq.ProjectListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	userID := utils.GetUserID(c)
	list, total, err := projectService.GetProjectList(req, userID)
	if err != nil {
		global.GVA_LOG.Error("获取项目列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "获取成功", c)
}

// SetProjectPermission
// @Tags      Dam
// @Summary   设置项目权限
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      exampleReq.ProjectPermissionRequest  true  "设置项目权限请求"
// @Success   200   {object}  response.Response{msg=string}  "设置成功"
// @Router    /project/permission/set [post]
func (p *ProjectApi) SetProjectPermission(c *gin.Context) {
	var req exampleReq.ProjectPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err := projectService.SetProjectPermission(req)
	if err != nil {
		global.GVA_LOG.Error("设置项目权限失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("设置成功", c)
}

// RemoveProjectPermission
// @Tags      Dam
// @Summary   移除项目权限
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      exampleReq.ProjectPermissionRequest  true  "移除项目权限请求"
// @Success   200   {object}  response.Response{msg=string}  "移除成功"
// @Router    /project/permission/remove [post]
func (p *ProjectApi) RemoveProjectPermission(c *gin.Context) {
	var req exampleReq.ProjectPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err := projectService.RemoveProjectPermission(req)
	if err != nil {
		global.GVA_LOG.Error("移除项目权限失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("移除成功", c)
}

// GetProjectPermissions
// @Tags      Dam
// @Summary   获取项目权限列表
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     projectId  query      string  true  "项目ID"
// @Success   200   {object}  response.Response{data=[][]string,msg=string}  "获取成功"
// @Router    /project/permission/list [get]
func (p *ProjectApi) GetProjectPermissions(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		response.FailWithMessage("项目ID不能为空", c)
		return
	}

	policies, err := projectService.GetProjectPermissions(projectID)
	if err != nil {
		global.GVA_LOG.Error("获取项目权限失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(policies, "获取成功", c)
}
