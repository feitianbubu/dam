package example

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example/request"
	exampleRes "github.com/flipped-aurora/gin-vue-admin/server/model/example/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
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
// @Param     projectId  formData  string                                                       false  "项目ID"
// @Param     tags  formData  string                                                         false  "自定义标签，用逗号分隔，如：tag1,tag2,tag3"
// @Param     metadata  formData  string                                                       false  "文件元数据，JSON格式字符串"
// @Param     autoMetadata  formData  bool                                           false  "是否自动生成元数据，默认false"
// @Param     waitForMetadata  formData  bool                                                   false  "是否等待metadata处理完成再返回，默认false（异步处理）"
// @Param     overwrite  formData  bool                                                   false  "是否覆盖同名文件，默认false（不覆盖则返回错误）"
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
	authorityID := utils.GetUserAuthorityId(c)

	// 构建业务元数据
	bizMetadata := example.BizMetadata{
		ProjectID: c.PostForm("projectId"),
		Tags:      parseTagsFromForm(c.PostForm("tags")),
	}

	// 获取metadata参数并尝试解析为JSON
	var fileMetadata *example.FileMetadata
	if metadataStr := c.PostForm("metadata"); metadataStr != "" {
		fileMetadata = &example.FileMetadata{}
		if err := json.Unmarshal([]byte(metadataStr), fileMetadata); err != nil {
			response.FailWithMessage(fmt.Sprintf("metadata参数格式错误，必须为有效的JSON字符串:%v", err), c)
			return
		}
	}

	// 获取是否自动生成元数据参数，默认为true
	autoMetadata := c.DefaultPostForm("autoMetadata", "false") == "true"

	// 获取是否等待metadata参数
	waitForMetadata := c.DefaultPostForm("waitForMetadata", "false") == "true"

	// 获取是否覆盖同名文件参数，默认为false
	overwrite := c.DefaultPostForm("overwrite", "false") == "true"

	// 记录上传用户信息
	global.GVA_LOG.Info("文件上传用户信息",
		zap.Uint("userID", userID),
		zap.String("userName", userName),
		zap.Uint("authorityID", authorityID),
		zap.String("projectId", bizMetadata.ProjectID),
		zap.Bool("autoMetadata", autoMetadata),
		zap.Bool("waitForMetadata", waitForMetadata),
		zap.Bool("overwrite", overwrite))

	// 构建上传选项
	uploadOpts := &example.FileUploadOptions{
		FileHeader:       header,
		NoSave:           noSave,
		ClassId:          classId,
		UserID:           userID,
		UserName:         userName,
		AuthorityID:      authorityID,
		BizMetadata:      &bizMetadata,
		ProvidedMetadata: fileMetadata,
		AutoMetadata:     autoMetadata,
		WaitForMetadata:  waitForMetadata,
		Overwrite:        overwrite,
	}

	// 调用服务层上传文件
	file, err = fileUploadAndDownloadService.UploadFileWithMetadata(uploadOpts)
	if err != nil {
		global.GVA_LOG.Error("上传文件失败!", zap.Error(err))
		response.FailWithMessage(fmt.Sprintf("上传文件失败: %v", err), c)
		return
	}

	// 根据处理状态返回不同的消息
	msg := "上传成功"
	if waitForMetadata && file.ProcessStatus == example.ProcessStatusCompleted {
		msg = "上传成功，文件处理完成"
	} else if waitForMetadata {
		msg = fmt.Sprintf("上传成功，文件处理状态: %s", file.ProcessStatus)
	}

	response.OkWithDetailed(exampleRes.ExaFileResponse{File: file}, msg, c)
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
	err := c.ShouldBindBodyWith(&file, binding.JSON)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 获取当前用户信息
	userID := utils.GetUserID(c)
	authorityID := utils.GetUserAuthorityId(c)

	err = fileUploadAndDownloadService.EditFileName(file, userID, authorityID)
	if err != nil {
		global.GVA_LOG.Error("编辑失败!", zap.Error(err))
		response.FailWithMessage("编辑失败: "+err.Error(), c)
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
	err := c.ShouldBindBodyWith(&file, binding.JSON)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 获取当前用户信息
	userID := utils.GetUserID(c)
	authorityID := utils.GetUserAuthorityId(c)

	if err := fileUploadAndDownloadService.DeleteFile(file, userID, authorityID); err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败: "+err.Error(), c)
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

	// 如果存储后端支持预签名URL，替换为预签名URL（1小时有效期）
	file.Url = fileUploadAndDownloadService.GetPresignedURL(&file, time.Hour)

	response.OkWithDetailed(file, "获取文件详情成功", c)
}

// GetFileList
// @Tags      Dam
// @Summary   搜索文件列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      request.ExaFileSearchRequest                                        true  "页码, 每页大小, 分类id, 项目id, 标签过滤, 用户名过滤, etag过滤, 可选的向量搜索参数"
// @Success   200   {object}  response.Response{data=response.PageResult,msg=string}  "分页文件列表,返回包括列表,总数,页码,每页数量"
// @Router    /fileUploadAndDownload/getFileList [post]
func (b *FileUploadAndDownloadApi) GetFileList(c *gin.Context) {
	var searchInfo request.ExaFileSearchRequest
	err := c.ShouldBindBodyWith(&searchInfo, binding.JSON)
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

	searchInfo.GetSearchDefaults()

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

	list, total, err := fileUploadAndDownloadService.GetFileRecordInfoListWithVectorFilter(searchInfo, vectorDocumentIDs)
	if err != nil {
		global.GVA_LOG.Error("文件搜索失败!", zap.Error(err))
		response.FailWithMessage("文件搜索失败: "+err.Error(), c)
		return
	}

	if searchInfo.IsVectorSearch() {
		list = fileUploadAndDownloadService.ApplyVectorScoresToResults(list, vectorScores)
		global.GVA_LOG.Info("文件搜索完成",
			zap.String("searchType", searchType),
			zap.Int("resultCount", len(list)))
	}

	// 为所有文件生成预签名URL（1小时有效期）
	fileUploadAndDownloadService.GetPresignedURLForFiles(list, time.Hour)

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
// @Success   200  {object}  response.Response{data=map[string]interface{},msg=string}  "重试文件处理，返回处理后的metadata"
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

	// 同步重试处理 - 使用通用重试逻辑
	err := fileUploadAndDownloadService.ProcessFileWithRetry(fileID, 3, "手动文件处理重试")
	if err != nil {
		global.GVA_LOG.Error("重试处理失败!", zap.Error(err))
		response.FailWithMessage(fmt.Sprintf("重试处理失败: %v", err), c)
		return
	}

	// 获取处理后的文件信息，包含metadata
	file, err := fileUploadAndDownloadService.FindFile(fileID)
	if err != nil {
		global.GVA_LOG.Error("获取处理后文件信息失败!", zap.Error(err))
		response.FailWithMessage(fmt.Sprintf("获取文件信息失败: %v", err), c)
		return
	}

	// 返回metadata内容作为处理结果
	if file.Metadata.IsEmpty() {
		global.GVA_LOG.Info("文件处理成功，但metadata为空", zap.Uint64("fileID", uint64(fileID)))
		response.OkWithDetailed(example.FileMetadata{}, "重试处理成功，但metadata为空", c)
	} else {
		global.GVA_LOG.Info("重试处理成功", zap.Uint64("fileID", uint64(fileID)), zap.String("metadata", file.Metadata.String()))
		response.OkWithDetailed(file.Metadata, "重试处理成功", c)
	}
}
