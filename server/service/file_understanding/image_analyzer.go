package file_understanding

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/sashabaranov/go-openai"
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
	return a.CheckFileTypeSupport(fileType, a.SupportedFileTypes())
}

// AnalyzeFile 分析图片文件
func (a *ImageAnalyzer) AnalyzeFile(ctx context.Context, filePath string, fileType string, modelConfig config.ModelConfig) (*example.FileMetadata, error) {
	if !a.IsEnabled() {
		a.LogDisabledSkip(filePath, "图片")
		return a.CreateDisabledResponse(filePath, fileType, "图片"), nil
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

	a.LogAnalysisStart(filePath, fileType, "图片", modelConfig)

	metadata, err := a.client.analyzeWithOpenAI(ctx, filePath, fileType, modelConfig, messageParts)
	if err != nil {
		return nil, err
	}

	// 为图片添加特定标签
	metadata.Tags = append(metadata.Tags, "image")
	metadata.Category = "image"

	a.LogAnalysisComplete(filePath, "图片", metadata.Description)

	return metadata, nil
}
