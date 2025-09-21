package file_understanding

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
)

// FileUnderstandingService 文件理解服务接口
type FileUnderstandingService interface {
	// AnalyzeFile 分析文件，返回文件元数据
	AnalyzeFile(filePath string, fileType string) (*example.FileMetadata, error)

	// GetSupportedFileTypes 获取支持的文件类型
	GetSupportedFileTypes() []string

	// IsFileTypeSupported 检查文件类型是否支持
	IsFileTypeSupported(fileType string) bool
}

// FileProcessor 文件处理器接口
type FileProcessor interface {
	// ProcessFile 处理文件，异步调用多模态API并更新数据库
	ProcessFile(fileID uint) error

	// RetryProcessFile 重试处理失败的文件
	RetryProcessFile(fileID uint) error

	// UpdateProcessStatus 更新文件处理状态
	UpdateProcessStatus(fileID uint, status string, metadata *example.FileMetadata, errorMsg string) error
}
