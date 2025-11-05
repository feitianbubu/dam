package example

import (
	"errors"
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example/request"
	exampleRes "github.com/flipped-aurora/gin-vue-admin/server/model/example/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UpdateFile
// @Tags      Dam
// @Summary   更新文件信息
// @Security  ApiKeyAuth
// @accept    multipart/form-data
// @Produce   application/json
// @Param     id  formData  uint  true  "文件ID"
// @Param     file  formData  file  false  "新文件（可选，如果提供则替换原文件内容）"
// @Param     name  formData  string  false  "文件名"
// @Param     classId  formData  int  false  "分类ID"
// @Param     projectId  formData  string  false  "项目ID"
// @Param     tags  formData  string  false  "标签，用逗号分隔"
// @Param     reprocess  formData  bool  false  "是否重新处理"
// @Param     updateVector  formData  bool  false  "是否更新向量"
// @Success   200   {object}  response.Response{data=exampleRes.ExaFileResponse,msg=string}  "更新文件信息成功，返回包括文件详情"
// @Router    /fileUploadAndDownload/update [put]
func (b *FileUploadAndDownloadApi) UpdateFile(c *gin.Context) {
	// 解析表单参数
	req, err := b.parseUpdateRequest(c)
	if err != nil {
		response.FailWithMessage("请求参数错误: "+err.Error(), c)
		return
	}

	// 检查是否有文件上传
	_, fileHeader, fileErr := c.Request.FormFile("file")

	// 准备更新选项
	opts := b.prepareUpdateOptions(req)

	// 如果提供了新文件，设置文件头
	if fileErr == nil && fileHeader != nil {
		opts.NewFile = fileHeader
	}

	// 获取当前用户ID并记录修改人
	userID := utils.GetUserID(c)
	opts.UpdateUserID = &userID

	// 调用服务层更新文件
	file, err := fileUploadAndDownloadService.UpdateFile(req.ID, opts)
	if err != nil {
		global.GVA_LOG.Error("更新文件失败!", zap.Error(err))
		response.FailWithMessage("更新文件失败: "+err.Error(), c)
		return
	}

	// 构建响应消息
	msg := b.buildUpdateMessage(opts.Reprocess, opts.UpdateVector, opts.NewFile != nil)
	response.OkWithDetailed(exampleRes.ExaFileResponse{File: file}, msg, c)
}

func (b *FileUploadAndDownloadApi) prepareUpdateOptions(req *request.ExaFileUpdateRequest) *example.FileUpdateOptions {
	opts := &example.FileUpdateOptions{
		Name:    req.Name,
		ClassId: req.ClassId,
	}

	// 处理业务元数据
	if bizMetadata := req.GetBizMetadata(); bizMetadata != nil {
		opts.BizMetadata = bizMetadata
	}

	// 处理布尔选项（默认为false）
	opts.Reprocess = req.Reprocess != nil && *req.Reprocess
	opts.UpdateVector = req.UpdateVector != nil && *req.UpdateVector

	return opts
}

// parseUpdateRequest 从表单中解析更新请求参数
func (b *FileUploadAndDownloadApi) parseUpdateRequest(c *gin.Context) (*request.ExaFileUpdateRequest, error) {
	req := &request.ExaFileUpdateRequest{}

	// 解析文件ID（必需）
	if idStr := c.PostForm("id"); idStr == "" {
		return nil, errors.New("文件ID不能为空")
	} else {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return nil, errors.New("文件ID格式错误")
		}
		req.ID = uint(id)
	}

	// 解析可选参数
	if name := c.PostForm("name"); name != "" {
		req.Name = &name
	}

	if classIdStr := c.PostForm("classId"); classIdStr != "" {
		classId, err := strconv.Atoi(classIdStr)
		if err != nil {
			return nil, errors.New("分类ID格式错误")
		}
		req.ClassId = &classId
	}

	if projectId := c.PostForm("projectId"); projectId != "" {
		req.ProjectID = &projectId
	}

	if tags := c.PostForm("tags"); tags != "" {
		req.Tags = &tags
	}

	if reprocessStr := c.PostForm("reprocess"); reprocessStr != "" {
		reprocess := reprocessStr == "true"
		req.Reprocess = &reprocess
	}

	if updateVectorStr := c.PostForm("updateVector"); updateVectorStr != "" {
		updateVector := updateVectorStr == "true"
		req.UpdateVector = &updateVector
	}

	return req, nil
}

// buildUpdateMessage 构建更新成功的响应消息
func (b *FileUploadAndDownloadApi) buildUpdateMessage(reprocess, updateVector, fileReplaced bool) string {
	msg := "更新成功"
	if fileReplaced {
		msg += "，文件内容已替换"
	}
	if reprocess {
		msg += "，文件已进入重新处理队列"
	}
	if updateVector {
		msg += "，向量化数据将异步更新"
	}
	return msg
}
