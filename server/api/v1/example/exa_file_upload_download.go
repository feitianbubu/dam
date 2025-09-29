package example

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonRequest "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example/request"
	exampleRes "github.com/flipped-aurora/gin-vue-admin/server/model/example/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FileUploadAndDownloadApi struct{}

// UploadFile
// @Tags      Dam
// @Summary   上传文件示例
// @Security  ApiKeyAuth
// @accept    multipart/form-data
// @Produce   application/json
// @Param     file  formData  file                                                           true  "上传文件示例"
// @Success   200   {object}  response.Response{data=exampleRes.ExaFileResponse,msg=string}  "上传文件示例,返回包括文件详情"
// @Router    /fileUploadAndDownload/upload [post]
func (b *FileUploadAndDownloadApi) UploadFile(c *gin.Context) {
	var file example.ExaFileUploadAndDownload
	noSave := c.DefaultQuery("noSave", "0")
	_, header, err := c.Request.FormFile("file")
	classId, _ := strconv.Atoi(c.DefaultPostForm("classId", "0"))
	if err != nil {
		global.GVA_LOG.Error("接收文件失败!", zap.Error(err))
		response.FailWithMessage("接收文件失败", c)
		return
	}
	file, err = fileUploadAndDownloadService.UploadFile(header, noSave, classId) // 文件上传后拿到文件路径
	if err != nil {
		global.GVA_LOG.Error("上传文件失败!", zap.Error(err))
		response.FailWithMessage("上传文件失败", c)
		return
	}
	response.OkWithDetailed(exampleRes.ExaFileResponse{File: file}, "上传成功", c)
}

// EditFileName 编辑文件名或者备注
func (b *FileUploadAndDownloadApi) EditFileName(c *gin.Context) {
	var file example.ExaFileUploadAndDownload
	err := c.ShouldBindJSON(&file)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = fileUploadAndDownloadService.EditFileName(file)
	if err != nil {
		global.GVA_LOG.Error("编辑失败!", zap.Error(err))
		response.FailWithMessage("编辑失败", c)
		return
	}
	response.OkWithMessage("编辑成功", c)
}

// DeleteFile
// @Tags      Dam
// @Summary   删除文件
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     data  body      request.ExaFileIDRequest  true  "传入文件里面id即可"
// @Success   200   {object}  response.Response{msg=string}     "删除文件"
// @Router    /fileUploadAndDownload/deleteFile [post]
func (b *FileUploadAndDownloadApi) DeleteFile(c *gin.Context) {
	var file example.ExaFileUploadAndDownload
	err := c.ShouldBindJSON(&file)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := fileUploadAndDownloadService.DeleteFile(file); err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// GetFileDetail
// @Tags      Dam
// @Summary   获取文件详情
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id  query      uint                                       true  "文件ID"
// @Success   200   {object}  response.Response{data=example.ExaFileUploadAndDownload,msg=string}  "文件详情,包含基本信息和AI分析元数据"
// @Router    /fileUploadAndDownload/getFileDetail [get]
func (b *FileUploadAndDownloadApi) GetFileDetail(c *gin.Context) {
	var fileIdQuery struct {
		ID uint `json:"id" form:"id"`
	}
	_ = c.ShouldBindQuery(&fileIdQuery)

	file, err := fileUploadAndDownloadService.FindFile(fileIdQuery.ID)
	if err != nil {
		global.GVA_LOG.Error("获取文件详情失败!", zap.Error(err))
		response.FailWithMessage("获取文件详情失败", c)
		return
	}
	response.OkWithDetailed(file, "获取文件详情成功", c)
}

// GetFileList
// @Tags      Dam
// @Summary   搜索文件列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      request.ExaFileSearchRequest                                        true  "页码, 每页大小, 分类id, 可选的向量搜索参数"
// @Success   200   {object}  response.Response{data=response.PageResult,msg=string}  "分页文件列表,返回包括列表,总数,页码,每页数量"
// @Router    /fileUploadAndDownload/getFileList [post]
func (b *FileUploadAndDownloadApi) GetFileList(c *gin.Context) {
	var searchInfo request.ExaFileSearchRequest
	err := c.ShouldBindJSON(&searchInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if searchInfo.Page <= 0 {
		searchInfo.Page = 1
	}
	if searchInfo.PageSize <= 0 {
		searchInfo.PageSize = 10
	}

	// 如果有向量搜索参数，使用新的搜索方法
	if searchInfo.IsVectorSearch() {
		list, total, err := fileUploadAndDownloadService.GetFileRecordInfoListWithSearch(searchInfo)
		if err != nil {
			global.GVA_LOG.Error("向量搜索失败!", zap.Error(err))
			response.FailWithMessage("向量搜索失败: "+err.Error(), c)
			return
		}
		response.OkWithDetailed(response.PageResult{
			List:     list,
			Total:    total,
			Page:     searchInfo.Page,
			PageSize: searchInfo.PageSize,
		}, "向量搜索成功", c)
		return
	}

	// 否则使用传统搜索方法（向后兼容）
	pageInfo := request.ExaAttachmentCategorySearch{
		ClassId: searchInfo.ClassId,
		PageInfo: commonRequest.PageInfo{
			Page:     searchInfo.Page,
			PageSize: searchInfo.PageSize,
			Keyword:  searchInfo.Keyword,
		},
	}
	list, total, err := fileUploadAndDownloadService.GetFileRecordInfoList(pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}

// ImportURL
// @Tags      ExaFileUploadAndDownload
// @Summary   导入URL
// @Security  ApiKeyAuth
// @Produce   application/json
// @Param     data  body      example.ExaFileUploadAndDownload  true  "对象"
// @Success   200   {object}  response.Response{msg=string}     "导入URL"
// @Router    /fileUploadAndDownload/importURL [post]
func (b *FileUploadAndDownloadApi) ImportURL(c *gin.Context) {
	var file []example.ExaFileUploadAndDownload
	err := c.ShouldBindJSON(&file)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := fileUploadAndDownloadService.ImportURL(&file); err != nil {
		global.GVA_LOG.Error("导入URL失败!", zap.Error(err))
		response.FailWithMessage("导入URL失败", c)
		return
	}
	response.OkWithMessage("导入URL成功", c)
}

// RetryFileProcessing
// @Tags      Dam
// @Summary   重试文件处理
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     id  query  int  true  "文件ID"
// @Success   200  {object}  response.Response{msg=string}  "重试文件处理"
// @Router    /fileUploadAndDownload/retryProcessing [post]
func (b *FileUploadAndDownloadApi) RetryFileProcessing(c *gin.Context) {
	var fileID uint
	if idStr := c.Query("id"); idStr == "" {
		response.FailWithMessage("文件ID不能为空", c)
		return
	} else if id, err := strconv.ParseUint(idStr, 10, 32); err != nil {
		response.FailWithMessage("文件ID格式错误", c)
		return
	} else {
		fileID = uint(id)
	}

	// 异步重试处理 - 使用通用重试逻辑
	go fileUploadAndDownloadService.ProcessFileWithRetry(fileID, 3, "手动文件处理重试")

	response.OkWithMessage("重试处理已启动，后台异步执行", c)
}
