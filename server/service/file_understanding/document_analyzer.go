package file_understanding

import (
	"context"
	"fmt"
	"slices"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// DocumentAnalyzer 文档文件分析器
type DocumentAnalyzer struct {
	*BaseAnalyzer
}

// NewDocumentAnalyzer 创建文档分析器
func NewDocumentAnalyzer(client *MultimodalAPIClient) *DocumentAnalyzer {
	return &DocumentAnalyzer{
		BaseAnalyzer: NewBaseAnalyzer(client),
	}
}

// SupportedFileTypes 返回支持的文档文件类型
func (a *DocumentAnalyzer) SupportedFileTypes() []string {
	return global.GVA_CONFIG.FileUnderstanding.SupportedFileTypes.Documents
}

// IsSupported 检查是否支持指定的文件类型
func (a *DocumentAnalyzer) IsSupported(fileType string) bool {
	return a.CheckFileTypeSupport(fileType, a.SupportedFileTypes())
}

// AnalyzeFile 分析文档文件
func (a *DocumentAnalyzer) AnalyzeFile(ctx context.Context, filePath string, fileType string, modelConfig config.ModelConfig) (*example.FileMetadata, error) {
	if !a.IsEnabled() {
		a.LogDisabledSkip(filePath, "文档")
		return a.CreateDisabledResponse(filePath, fileType, "文档"), nil
	}

	// 统一读取文档内容（无论是本地文件还是HTTP URL）
	content, err := a.client.readFileContent(filePath)
	if err != nil {
		global.GVA_LOG.Error("读取文档文件失败",
			zap.String("filePath", filePath),
			zap.Error(err))
		return nil, fmt.Errorf("读取文档文件失败: %w", err)
	}

	// 限制内容长度避免token过多
	if len(content) > 80000 {
		content = content[:80000] + "..."
	}

	prompt := `请分析这个文档文件，返回文档的主要内容概述, 请确保返回有用的、准确的分析结果`
	messageParts := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: fmt.Sprintf("%s\n\n文档内容：\n%s", prompt, content),
		},
	}

	a.LogAnalysisStart(filePath, fileType, "文档", modelConfig)
	global.GVA_LOG.Info("文档内容长度", zap.Int("contentLength", len(content)))

	metadata := example.FileMetadata{}

	if !slices.Contains([]string{"txt", "ini", "md", "log"}, fileType) {
		aiMetadata, err := a.client.analyzeWithOpenAI(ctx, filePath, fileType, modelConfig, messageParts)
		if err != nil {
			return nil, err
		}
		// 从 aiMetadata 中提取描述
		if desc, err := aiMetadata.GetString("description"); err == nil {
			content = desc
		}
	}
	_ = metadata.Set("description", content)
	_ = metadata.Set("contentType", fileType)
	_ = metadata.Set("category", "document")

	// 为文档添加特定标签
	_ = metadata.Set("tags", []string{"document", fileType})

	// 获取描述用于日志
	description, _ := metadata.GetString("description")
	a.LogAnalysisComplete(filePath, "文档", description)

	return &metadata, nil
}
