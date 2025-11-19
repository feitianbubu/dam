package example

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example/request"
	exampleRes "github.com/flipped-aurora/gin-vue-admin/server/model/example/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
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
// @Param     publicRead  formData  bool                                                   false  "是否允许公开读，默认false（私有）"
// @Success   200   {object}  response.Response{data=exampleRes.ExaFileResponse,msg=string}  "上传文件示例,返回包括文件详情"
// @Router    /fileUploadAndDownload/upload [post]
func (b *FileUploadAndDownloadApi) UploadFile(c *gin.Context) {
	var file example.ExaFileUploadAndDownload
	noSave := c.DefaultQuery("noSave", "0")
	_, header, err := c.Request.FormFile("file")
	classId, _ := strconv.Atoi(c.DefaultPostForm("classId", "0"))
	if classId == 0 {
		classId = 1
	}
	if err != nil {
		response.FailWithMessage("接收文件失败", c)
		return
	}

	userID := utils.GetUserID(c)
	userName := utils.GetUserName(c)
	authorityID := utils.GetUserAuthorityId(c)

	var fileMetadata *example.FileMetadata
	if metadataStr := c.PostForm("metadata"); metadataStr != "" {
		fileMetadata = &example.FileMetadata{}
		if err := json.Unmarshal([]byte(metadataStr), fileMetadata); err != nil {
			response.FailWithMessage(fmt.Sprintf("metadata参数格式错误: %v", err), c)
			return
		}
	}

	autoMetadata := c.DefaultPostForm("autoMetadata", "false") == "true"
	waitForMetadata := c.DefaultPostForm("waitForMetadata", "false") == "true"
	overwrite := c.DefaultPostForm("overwrite", "false") == "true"

	// 处理 publicRead 参数：区分"未传参数"和"明确传了 false"
	var publicRead *bool
	if publicReadStr := c.PostForm("publicRead"); publicReadStr != "" {
		value := publicReadStr == "true"
		publicRead = &value
	}

	uploadOpts := &example.FileUploadOptions{
		FileHeader:  header,
		NoSave:      noSave,
		ClassId:     classId,
		UserID:      userID,
		UserName:    userName,
		AuthorityID: authorityID,
		BizMetadata: &example.BizMetadata{
			ProjectID: c.PostForm("projectId"),
			Tags:      parseTagsFromForm(c.PostForm("tags")),
		},
		ProvidedMetadata: fileMetadata,
		AutoMetadata:     autoMetadata,
		WaitForMetadata:  waitForMetadata,
		Overwrite:        overwrite,
		PublicRead:       publicRead,
	}

	file, err = fileUploadAndDownloadService.UploadFileWithMetadata(uploadOpts)
	if err != nil {
		response.FailWithMessage(fmt.Sprintf("上传文件失败: %v", err), c)
		return
	}

	msg := "上传成功"
	if waitForMetadata {
		if file.ProcessStatus == example.ProcessStatusCompleted {
			msg = "上传成功，文件处理完成"
		} else {
			msg = fmt.Sprintf("上传成功，文件处理状态: %s", file.ProcessStatus)
		}
	}

	response.OkWithDetailed(exampleRes.ExaFileResponse{File: file}, msg, c)
}

func parseTagsFromForm(tagsStr string) []string {
	if tagsStr == "" {
		return []string{}
	}

	tags := strings.Split(tagsStr, ",")
	var parsedTags []string
	for _, tag := range tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			parsedTags = append(parsedTags, tag)
		}
	}
	return parsedTags
}

func (b *FileUploadAndDownloadApi) EditFileName(c *gin.Context) {
	var file example.ExaFileUploadAndDownload
	if err := c.ShouldBindBodyWith(&file, binding.JSON); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	userID := utils.GetUserID(c)
	authorityID := utils.GetUserAuthorityId(c)

	if err := fileUploadAndDownloadService.EditFileName(file, userID, authorityID); err != nil {
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
	if err := c.ShouldBindBodyWith(&file, binding.JSON); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	userID := utils.GetUserID(c)
	authorityID := utils.GetUserAuthorityId(c)

	if err := fileUploadAndDownloadService.DeleteFile(file, userID, authorityID); err != nil {
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
		response.FailWithMessage("获取文件详情失败", c)
		return
	}

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
	if err := c.ShouldBindBodyWith(&searchInfo, binding.JSON); err != nil {
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
	searchType := "传统搜索"

	if searchInfo.IsVectorSearch() {
		searchType = "向量搜索"

		vectorResult, err := fileUploadAndDownloadService.SearchVectorDocuments(
			searchInfo.Prompt,
			searchInfo.KnowledgeIDs,
			searchInfo.TopK,
			searchInfo.MinScore,
			searchInfo.SearchType,
		)
		if err != nil {
			response.FailWithMessage("向量搜索失败: "+err.Error(), c)
			return
		}

		if len(vectorResult.DocumentIDs) == 0 {
			response.OkWithDetailed(response.PageResult{
				List:     []example.ExaFileUploadAndDownload{},
				Total:    0,
				Page:     searchInfo.Page,
				PageSize: searchInfo.PageSize,
			}, "向量搜索完成，无匹配结果", c)
			return
		}

		vectorDocumentIDs = vectorResult.DocumentIDs
		vectorScores = vectorResult.Scores
	}

	list, total, err := fileUploadAndDownloadService.GetFileRecordInfoListWithVectorFilter(searchInfo, vectorDocumentIDs)
	if err != nil {
		response.FailWithMessage("文件搜索失败: "+err.Error(), c)
		return
	}

	if searchInfo.IsVectorSearch() {
		list = fileUploadAndDownloadService.ApplyVectorScoresToResults(list, vectorScores)
	}

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
	if err := c.ShouldBindJSON(&file); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if err := fileUploadAndDownloadService.ImportURL(&file); err != nil {
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

	if err := fileUploadAndDownloadService.ProcessFileWithRetry(fileID, 3, "手动文件处理重试"); err != nil {
		response.FailWithMessage(fmt.Sprintf("重试处理失败: %v", err), c)
		return
	}

	file, err := fileUploadAndDownloadService.FindFile(fileID)
	if err != nil {
		response.FailWithMessage(fmt.Sprintf("获取文件信息失败: %v", err), c)
		return
	}

	if file.Metadata.IsEmpty() {
		response.OkWithDetailed(example.FileMetadata{}, "重试处理成功，但metadata为空", c)
	} else {
		response.OkWithDetailed(file.Metadata, "重试处理成功", c)
	}
}
