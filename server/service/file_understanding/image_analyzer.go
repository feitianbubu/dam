package file_understanding

import (
	"context"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// ImageAnalyzer 图片文件分析器
type ImageAnalyzer struct {
	*BaseAnalyzer
}

// NewImageAnalyzer 创建图片分析器
func NewImageAnalyzer(client *MultimodalAPIClient) *ImageAnalyzer {
	return &ImageAnalyzer{
		BaseAnalyzer: NewBaseAnalyzer(client),
	}
}

// SupportedFileTypes 返回支持的图片文件类型
func (a *ImageAnalyzer) SupportedFileTypes() []string {
	return global.GVA_CONFIG.FileUnderstanding.SupportedFileTypes.Images
}

// IsSupported 检查是否支持指定的文件类型
func (a *ImageAnalyzer) IsSupported(fileType string) bool {
	supportedTypes := a.SupportedFileTypes()
	fileType = strings.ToLower(fileType)
	for _, t := range supportedTypes {
		if strings.ToLower(t) == fileType {
			return true
		}
	}
	return false
}

// AnalyzeFile 分析图片文件
func (a *ImageAnalyzer) AnalyzeFile(ctx context.Context, filePath string, fileType string, modelConfig config.ModelConfig) (*example.FileMetadata, error) {
	if !a.IsEnabled() {
		global.GVA_LOG.Info("图片分析器未启用，跳过处理", zap.String("filePath", filePath))
		return &example.FileMetadata{
			Description:  "图片分析功能未启用",
			ContentType:  fileType,
			DetectedText: "",
			Objects:      []string{},
			Tags:         []string{"skipped", "image"},
			FileSize:     0,
			Language:     "unknown",
			Category:     "image",
			Confidence:   0.0,
			ExtraData:    make(map[string]interface{}),
		}, nil
	}

	prompt := `请详细分析这张图片，并以JSON格式返回以下信息：
1. description: 图片的详细描述（中文）
2. detectedText: 图片中的文字内容（如果有）
3. objects: 检测到的物体/对象列表
4. tags: 相关标签列表
5. category: 图片类别（如：人物、风景、产品、文档等）
6. language: 如果有文字，检测到的语言
7. confidence: 分析的置信度（0-1之间的数值）

请确保返回有用的、准确的分析结果。`

	messageParts := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: prompt,
		},
		{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: filePath,
			},
		},
	}

	global.GVA_LOG.Info("开始分析图片文件",
		zap.String("filePath", filePath),
		zap.String("fileType", fileType),
		zap.String("model", modelConfig.Model))

	metadata, err := a.client.analyzeWithOpenAI(ctx, filePath, fileType, modelConfig, messageParts)
	if err != nil {
		return nil, err
	}

	// 为图片添加特定标签
	metadata.Tags = append(metadata.Tags, "image")
	metadata.Category = "image"

	global.GVA_LOG.Info("图片分析完成",
		zap.String("filePath", filePath),
		zap.String("description", metadata.Description))

	return metadata, nil
}
