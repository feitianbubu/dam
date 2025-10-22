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

	prompt := `请详细分析这张图片，返回图片的详细描述(中文), 只返回描述, 不要包含其他信息`

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
	var tags []string
	if err := metadata.Get("tags", &tags); err == nil {
		tags = append(tags, "image")
		_ = metadata.Set("tags", tags)
	} else {
		_ = metadata.Set("tags", []string{"image"})
	}
	_ = metadata.Set("category", "image")

	// 获取描述用于日志
	description, _ := metadata.GetString("description")
	a.LogAnalysisComplete(filePath, "图片", description)

	return metadata, nil
}
