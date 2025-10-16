package example

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example/request"
	exampleRes "github.com/flipped-aurora/gin-vue-admin/server/model/example/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
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
// @Param     classId  formData  int                                                            false  "分类ID，默认为1"
// @Param     tags  formData  string                                                         false  "自定义标签，用逗号分隔，如：tag1,tag2,tag3"
// @Param     username  formData  string                                                       false  "用户名，留空则使用当前登录用户名"
// @Param     remarks  formData  string                                                       false  "备注信息"
// @Success   200   {object}  response.Response{data=exampleRes.ExaFileResponse,msg=string}  "上传文件示例,返回包括文件详情"
// @Router    /fileUploadAndDownload/upload [post]
func (b *FileUploadAndDownloadApi) UploadFile(c *gin.Context) {
	var file example.ExaFileUploadAndDownload
	noSave := c.DefaultQuery("noSave", "0")
	_, header, err := c.Request.FormFile("file")
	classId, _ := strconv.Atoi(c.DefaultPostForm("classId", "0"))
	if classId == 0 {
		classId = 1 // 默认放在默认分类
	}
	if err != nil {
		global.GVA_LOG.Error("接收文件失败!", zap.Error(err))
		response.FailWithMessage("接收文件失败", c)
		return
	}

	// 获取当前用户信息
	userID := utils.GetUserID(c)
	userName := utils.GetUserName(c)

	// 解析业务元数据参数，优先使用表单提供的username，否则使用当前用户
	usernameValue := c.PostForm("username")
	if usernameValue == "" {
		// 如果表单中没有提供username，使用当前用户名
		usernameValue = userName
	}

	bizMetadata := example.BizMetadata{
		Tags:     parseTagsFromForm(c.PostForm("tags")),
		Username: usernameValue,
		Remarks:  c.PostForm("remarks"),
	}

	// 记录上传用户信息
	global.GVA_LOG.Info("文件上传用户信息",
		zap.Uint("userID", userID),
		zap.String("userName", userName),
		zap.String("fileUsername", usernameValue))

	file, err = fileUploadAndDownloadService.UploadFileWithMetadata(header, noSave, classId, &bizMetadata) // 文件上传后拿到文件路径
	if err != nil {
		global.GVA_LOG.Error("上传文件失败!", zap.Error(err))
		response.FailWithMessage(fmt.Sprintf("上传文件失败: %v", err), c)
		return
	}
	response.OkWithDetailed(exampleRes.ExaFileResponse{File: file}, "上传成功", c)
}

// parseTagsFromForm 解析表单中的tags字符串为标签数组
func parseTagsFromForm(tagsStr string) []string {
	if tagsStr == "" {
		return []string{}
	}

	tags := strings.Split(tagsStr, ",")
	var parsedTags []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			parsedTags = append(parsedTags, tag)
		}
	}

	return parsedTags
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
// @Param     data  body      request.ExaFileSearchRequest                                        true  "页码, 每页大小, 分类id, 标签过滤, 用户名过滤, 可选的向量搜索参数"
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

	// 执行搜索（支持向量搜索和传统搜索）
	var vectorDocumentIDs []string
	var vectorScores map[uint]float64
	var searchType string

	// 如果有向量搜索参数，先执行向量搜索获取文档ID
	if searchInfo.IsVectorSearch() {
		searchType = "向量搜索"

		// 执行向量搜索，获取匹配的文档ID和分数
		vectorResult, err := fileUploadAndDownloadService.SearchVectorDocuments(
			searchInfo.Prompt,
			searchInfo.KnowledgeIDs,
			searchInfo.TopK,
			searchInfo.MinScore,
			searchInfo.SearchType,
		)
		if err != nil {
			global.GVA_LOG.Error("向量搜索失败!", zap.Error(err))
			response.FailWithMessage("向量搜索失败: "+err.Error(), c)
			return
		}

		if len(vectorResult.DocumentIDs) == 0 {
			// 没有匹配的文档，返回空结果
			global.GVA_LOG.Info("向量搜索无匹配结果", zap.String("prompt", searchInfo.Prompt))
			response.OkWithDetailed(response.PageResult{
				List:     []example.ExaFileUploadAndDownload{},
				Total:    0,
				Page:     searchInfo.Page,
				PageSize: searchInfo.PageSize,
			}, "向量搜索完成，无匹配结果", c)
			return
		}

		// 保存向量搜索结果
		vectorDocumentIDs = vectorResult.DocumentIDs
		vectorScores = vectorResult.Scores

		global.GVA_LOG.Info("向量搜索完成",
			zap.String("prompt", searchInfo.Prompt),
			zap.Int("matchedDocuments", len(vectorDocumentIDs)))
	} else {
		searchType = "传统搜索"
	}

	// 使用统一的搜索方法执行文件搜索
	list, total, err := fileUploadAndDownloadService.GetFileRecordInfoListWithVectorFilter(searchInfo, vectorDocumentIDs)
	if err != nil {
		global.GVA_LOG.Error("文件搜索失败!", zap.Error(err))
		response.FailWithMessage("文件搜索失败: "+err.Error(), c)
		return
	}

	// 如果是向量搜索，应用分数并排序
	if searchInfo.IsVectorSearch() {
		list = fileUploadAndDownloadService.ApplyVectorScoresToResults(list, vectorScores)
		global.GVA_LOG.Info("文件搜索完成",
			zap.String("searchType", searchType),
			zap.Int("resultCount", len(list)))
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     searchInfo.Page,
		PageSize: searchInfo.PageSize,
	}, searchType+"成功", c)
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
