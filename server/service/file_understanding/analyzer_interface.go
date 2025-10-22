package file_understanding

import (
	"context"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"go.uber.org/zap"
)

type FileAnalyzer interface {
	AnalyzeFile(ctx context.Context, filePath string, fileType string, modelConfig config.ModelConfig) (*example.FileMetadata, error)

	SupportedFileTypes() []string

	IsSupported(fileType string) bool
}

type BaseAnalyzer struct {
	client  *MultimodalAPIClient
	enabled bool
}

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

// CheckFileTypeSupport 通用的文件类型支持检查方法
func (b *BaseAnalyzer) CheckFileTypeSupport(fileType string, supportedTypes []string) bool {
	fileType = strings.ToLower(fileType)
	for _, t := range supportedTypes {
		if strings.ToLower(t) == fileType {
			return true
		}
	}
	return false
}

// CreateDisabledResponse 创建分析器未启用时的默认响应
func (b *BaseAnalyzer) CreateDisabledResponse(filePath, fileType, analyzerName string) *example.FileMetadata {
	return &example.FileMetadata{}
}

// LogAnalysisStart 记录分析开始日志
func (b *BaseAnalyzer) LogAnalysisStart(filePath, fileType, analyzerName string, modelConfig config.ModelConfig) {
	global.GVA_LOG.Info("开始分析"+analyzerName+"文件",
		zap.String("filePath", filePath),
		zap.String("fileType", fileType),
		zap.String("model", modelConfig.Model))
}

// LogAnalysisComplete 记录分析完成日志
func (b *BaseAnalyzer) LogAnalysisComplete(filePath, analyzerName, description string) {
	global.GVA_LOG.Info(analyzerName+"分析完成",
		zap.String("filePath", filePath),
		zap.String("description", description))
}

// LogDisabledSkip 记录分析器未启用跳过处理的日志
func (b *BaseAnalyzer) LogDisabledSkip(filePath, analyzerName string) {
	global.GVA_LOG.Info(analyzerName+"分析器未启用，跳过处理", zap.String("filePath", filePath))
}
