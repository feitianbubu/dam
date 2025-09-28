package file_understanding

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// AudioAnalyzer 音频文件分析器
type AudioAnalyzer struct {
	*BaseAnalyzer
}

// NewAudioAnalyzer 创建音频分析器
func NewAudioAnalyzer(client *MultimodalAPIClient) *AudioAnalyzer {
	return &AudioAnalyzer{
		BaseAnalyzer: NewBaseAnalyzer(client),
	}
}

// SupportedFileTypes 返回支持的音频文件类型
func (a *AudioAnalyzer) SupportedFileTypes() []string {
	return global.GVA_CONFIG.FileUnderstanding.SupportedFileTypes.Audios
}

// IsSupported 检查是否支持指定的文件类型
func (a *AudioAnalyzer) IsSupported(fileType string) bool {
	return a.CheckFileTypeSupport(fileType, a.SupportedFileTypes())
}

// AnalyzeFile 分析音频文件
func (a *AudioAnalyzer) AnalyzeFile(ctx context.Context, filePath string, fileType string, modelConfig config.ModelConfig) (*example.FileMetadata, error) {
	if !a.IsEnabled() {
		a.LogDisabledSkip(filePath, "音频")
		return a.CreateDisabledResponse(filePath, fileType, "音频"), nil
	}

	// 处理文件路径：如果是URL则下载到临时文件
	localFilePath := filePath
	var tempFile *os.File
	var err error

	if strings.HasPrefix(filePath, "http") {
		// 下载URL文件到临时位置
		tempFile, err = a.client.downloadAudioFile(ctx, filePath, fileType)
		if err != nil {
			return nil, fmt.Errorf("下载音频文件失败: %w", err)
		}
		defer func() {
			if closeErr := tempFile.Close(); closeErr != nil {
				global.GVA_LOG.Error("关闭临时文件失败", zap.Error(closeErr))
			}
			if removeErr := os.Remove(tempFile.Name()); removeErr != nil {
				global.GVA_LOG.Error("删除临时文件失败", zap.Error(removeErr))
			}
		}()
		localFilePath = tempFile.Name()
	}

	// 使用配置的模型进行音频转录
	audioModel := modelConfig.Model
	if audioModel == "" {
		audioModel = openai.Whisper1 // 默认Whisper模型
	}

	req := openai.AudioRequest{
		Model:    audioModel,
		FilePath: localFilePath,
	}

	a.LogAnalysisStart(filePath, fileType, "音频", modelConfig)
	global.GVA_LOG.Info("本地文件路径", zap.String("localPath", localFilePath), zap.String("model", audioModel))

	resp, err := a.client.client.CreateTranscription(ctx, req)
	if err != nil {
		global.GVA_LOG.Error("OpenAI Whisper API调用失败", zap.Error(err))
		return nil, fmt.Errorf("OpenAI Whisper API调用失败: %w", err)
	}

	global.GVA_LOG.Info("音频转录完成",
		zap.String("filePath", filePath),
		zap.String("text", resp.Text),
		zap.String("language", resp.Language))

	// 分析转录文本内容并生成描述
	var description string
	var tags []string
	var objects []string

	if resp.Text != "" {
		description = fmt.Sprintf("音频转录内容：%s", resp.Text)
		tags = append(tags, "语音", "转录")
		objects = append(objects, "语音内容")

		// 根据内容长度判断音频类型
		if len(resp.Text) > 100 {
			tags = append(tags, "长语音")
		} else {
			tags = append(tags, "短语音")
		}
	} else {
		description = "音频文件，未检测到语音内容，可能为音乐或环境声音"
		tags = append(tags, "非语音音频")
		objects = append(objects, "背景音", "音效")
	}

	// 获取文件大小
	var fileSize int64
	if fileInfo, err := os.Stat(localFilePath); err == nil {
		fileSize = fileInfo.Size()
	}

	// 构建FileMetadata
	metadata := &example.FileMetadata{
		Description:  description,
		ContentType:  fileType,
		DetectedText: resp.Text,
		Objects:      objects,
		Tags:         append(tags, "audio", fileType),
		FileSize:     fileSize,
		Language:     resp.Language,
		Category:     "audio",
		Confidence:   0.9, // Whisper API通常有较高的准确度
		ExtraData: map[string]interface{}{
			"model":       audioModel,
			"duration":    resp.Duration,
			"segments":    resp.Segments,
			"rawResponse": resp.Text,
		},
	}

	a.LogAnalysisComplete(filePath, "音频", description)
	global.GVA_LOG.Info("音频转录详情", zap.String("language", resp.Language))

	return metadata, nil
}
