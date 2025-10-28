package example

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example/request"
	exampleRes "github.com/flipped-aurora/gin-vue-admin/server/model/example/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UpdateFile
// @Tags      Dam
// @Summary   更新文件信息
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      request.ExaFileUpdateRequest  true  "更新文件信息请求" SchemaExample({\"id\":1,\"name\":\"示例文件.jpg\",\"classId\":1,\"projectId\":\"proj_123456\",\"metadata\":\"{\\\"width\\\":1920,\\\"height\\\":1080}\",\"tags\":\"cinx,test\",\"reprocess\":false,\"updateVector\":false})
// @Success   200   {object}  response.Response{data=exampleRes.ExaFileResponse,msg=string}  "更新文件信息成功，返回包括文件详情"
// @Router    /fileUploadAndDownload/update [put]
func (b *FileUploadAndDownloadApi) UpdateFile(c *gin.Context) {
	var req request.ExaFileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数错误: "+err.Error(), c)
		return
	}

	// 准备更新选项
	opts := b.prepareUpdateOptions(&req)

	// 调用服务层更新文件
	file, err := fileUploadAndDownloadService.UpdateFile(req.ID, opts)
	if err != nil {
		global.GVA_LOG.Error("更新文件失败!", zap.Error(err))
		response.FailWithMessage("更新文件失败: "+err.Error(), c)
		return
	}

	// 构建响应消息
	msg := b.buildUpdateMessage(opts.Reprocess, opts.UpdateVector)
	response.OkWithDetailed(exampleRes.ExaFileResponse{File: file}, msg, c)
}

func (b *FileUploadAndDownloadApi) prepareUpdateOptions(req *request.ExaFileUpdateRequest) *example.FileUpdateOptions {
	opts := &example.FileUpdateOptions{
		//Name:    req.Name,
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

// buildUpdateMessage 构建更新成功的响应消息
func (b *FileUploadAndDownloadApi) buildUpdateMessage(reprocess, updateVector bool) string {
	msg := "更新成功"
	if reprocess {
		msg += "，文件已进入重新处理队列"
	}
	if updateVector {
		msg += "，向量化数据将异步更新"
	}
	return msg
}
