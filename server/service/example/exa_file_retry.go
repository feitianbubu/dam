package example

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/service/file_understanding"
	"go.uber.org/zap"
)

// ProcessFileWithRetry 通用的文件处理重试逻辑
// fileID: 文件ID
// maxRetries: 最大重试次数
// context: 处理上下文，用于日志标识
func (e *FileUploadAndDownloadService) ProcessFileWithRetry(fileID uint, maxRetries int, context string) {
	global.GVA_LOG.Info("开始文件处理重试",
		zap.String("context", context),
		zap.Uint64("fileID", uint64(fileID)),
		zap.Int("maxRetries", maxRetries))

	var processErr error

	// 创建文件处理器
	fileUnderstandingService := file_understanding.NewMultimodalAPIClient()
	processor := file_understanding.NewFileProcessor(fileUnderstandingService)

	for i := 0; i <= maxRetries; i++ {
		if processErr = processor.ProcessFile(fileID); processErr == nil {
			global.GVA_LOG.Info("文件处理成功",
				zap.String("context", context),
				zap.Uint64("fileID", uint64(fileID)))
			return
		}

		global.GVA_LOG.Warn("文件处理失败，准备重试",
			zap.String("context", context),
			zap.Uint64("fileID", uint64(fileID)),
			zap.Int("重试次数", i),
			zap.Error(processErr))

		if i < maxRetries {
			time.Sleep(time.Duration(i+1) * time.Second * 2) // 指数退避
		}
	}

	global.GVA_LOG.Error("文件处理最终失败",
		zap.String("context", context),
		zap.Uint64("fileID", uint64(fileID)),
		zap.Error(processErr))
}
