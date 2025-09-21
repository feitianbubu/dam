package file_understanding

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
)

// FileAnalyzer 文件分析器接口
type FileAnalyzer interface {
	// AnalyzeFile 分析文件并返回元数据
	AnalyzeFile(ctx context.Context, filePath string, fileType string, modelConfig config.ModelConfig) (*example.FileMetadata, error)

	// SupportedFileTypes 返回支持的文件类型
	SupportedFileTypes() []string

	// IsSupported 检查是否支持指定的文件类型
	IsSupported(fileType string) bool
}

// BaseAnalyzer 基础分析器，包含公共逻辑
type BaseAnalyzer struct {
	client  *MultimodalAPIClient
	enabled bool
}

// NewBaseAnalyzer 创建基础分析器
func NewBaseAnalyzer(client *MultimodalAPIClient) *BaseAnalyzer {
	return &BaseAnalyzer{
		client:  client,
		enabled: client != nil && client.enabled,
	}
}

// IsEnabled 检查分析器是否启用
func (b *BaseAnalyzer) IsEnabled() bool {
	return b.enabled
}

// GetClient 获取多模态客户端
func (b *BaseAnalyzer) GetClient() *MultimodalAPIClient {
	return b.client
}
