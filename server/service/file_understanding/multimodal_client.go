package file_understanding

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// MultimodalAPIClient OpenAI多模态API客户端
type MultimodalAPIClient struct {
	client    *openai.Client
	timeout   time.Duration
	enabled   bool
	analyzers map[string]FileAnalyzer
}

func NewMultimodalAPIClient() *MultimodalAPIClient {
	fuConfig := global.GVA_CONFIG.FileUnderstanding

	if !fuConfig.Enabled || fuConfig.APIKey == "" || fuConfig.APIKey == "your-openai-api-key-here" {
		global.GVA_LOG.Warn("文件理解功能未启用或配置不完整",
			zap.Bool("enabled", fuConfig.Enabled),
			zap.String("apiKey", "***masked***"),
			zap.String("endpoint", fuConfig.APIEndpoint))
		return &MultimodalAPIClient{enabled: false}
	}

	clientConfig := openai.DefaultConfig(fuConfig.APIKey)
	if fuConfig.APIEndpoint != "" {
		clientConfig.BaseURL = fuConfig.APIEndpoint
	}
	if !strings.HasSuffix(clientConfig.BaseURL, "/v1") {
		clientConfig.BaseURL = strings.TrimRight(clientConfig.BaseURL, "/") + "/v1"
	}

	global.GVA_LOG.Info("初始化OpenAI客户端",
		zap.String("endpoint", clientConfig.BaseURL))

	client := openai.NewClientWithConfig(clientConfig)

	timeout := time.Duration(fuConfig.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // 默认30秒超时
	}

	mac := &MultimodalAPIClient{
		client:    client,
		timeout:   timeout,
		enabled:   true,
		analyzers: make(map[string]FileAnalyzer),
	}

	// 初始化各类型分析器
	mac.analyzers["image"] = NewImageAnalyzer(mac)
	mac.analyzers["document"] = NewDocumentAnalyzer(mac)
	mac.analyzers["audio"] = NewAudioAnalyzer(mac)
	mac.analyzers["video"] = NewVideoAnalyzer(mac)

	return mac
}

func (c *MultimodalAPIClient) AnalyzeFile(filePath string, fileType string) (*example.FileMetadata, error) {
	if !c.enabled {
		global.GVA_LOG.Info("文件理解功能未启用，跳过处理", zap.String("filePath", filePath))
		// 返回空的 FileMetadata
		metadata := example.FileMetadata{}
		_ = metadata.FromMap(map[string]interface{}{
			"description": fmt.Sprintf("文件理解功能未启用 - %s", fileType),
			"contentType": fileType,
			"tags":        []string{"skipped"},
			"category":    "unknown",
		})
		return &metadata, nil
	}

	if !c.IsFileTypeSupported(fileType) {
		return nil, fmt.Errorf("不支持的文件类型: %s", fileType)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	category := c.getCategoryByFileType(fileType)
	modelConfig := global.GVA_CONFIG.FileUnderstanding.GetModelConfigForFileType(category)
	global.GVA_LOG.Info("开始调用OpenAI API分析文件",
		zap.String("filePath", filePath),
		zap.String("fileType", fileType),
		zap.String("category", category),
		zap.String("model", modelConfig.Model),
		zap.Int("maxTokens", modelConfig.MaxTokens),
		zap.Float32("temperature", modelConfig.Temperature))

	analyzer, exists := c.analyzers[category]
	if !exists || !analyzer.IsSupported(fileType) {
		return nil, fmt.Errorf("暂不支持分析此类型文件: %s (category: %s)", fileType, category)
	}

	return analyzer.AnalyzeFile(ctx, filePath, fileType, modelConfig)
}

// GetSupportedFileTypes 获取支持的文件类型
func (c *MultimodalAPIClient) GetSupportedFileTypes() []string {
	supportedTypes := global.GVA_CONFIG.FileUnderstanding.SupportedFileTypes
	var allTypes []string

	// 合并所有支持的文件类型
	allTypes = append(allTypes, supportedTypes.Images...)
	allTypes = append(allTypes, supportedTypes.Documents...)
	allTypes = append(allTypes, supportedTypes.Videos...)
	allTypes = append(allTypes, supportedTypes.Audios...)

	return allTypes
}

// IsFileTypeSupported 检查文件类型是否支持
func (c *MultimodalAPIClient) IsFileTypeSupported(fileType string) bool {
	supportedTypes := c.GetSupportedFileTypes()
	for _, t := range supportedTypes {
		if strings.ToLower(t) == strings.ToLower(fileType) {
			return true
		}
	}
	return false
}

// isImageFile 判断是否为图片文件
func (c *MultimodalAPIClient) isImageFile(fileType string) bool {
	imageTypes := global.GVA_CONFIG.FileUnderstanding.SupportedFileTypes.Images
	fileType = strings.ToLower(fileType)
	for _, t := range imageTypes {
		if strings.ToLower(t) == fileType {
			return true
		}
	}
	return false
}

// isDocumentFile 判断是否为文档文件
func (c *MultimodalAPIClient) isDocumentFile(fileType string) bool {
	documentTypes := global.GVA_CONFIG.FileUnderstanding.SupportedFileTypes.Documents
	fileType = strings.ToLower(fileType)
	for _, t := range documentTypes {
		if strings.ToLower(t) == fileType {
			return true
		}
	}
	return false
}

// isAudioFile 判断是否为音频文件
func (c *MultimodalAPIClient) isAudioFile(fileType string) bool {
	audioTypes := global.GVA_CONFIG.FileUnderstanding.SupportedFileTypes.Audios
	fileType = strings.ToLower(fileType)
	for _, t := range audioTypes {
		if strings.ToLower(t) == fileType {
			return true
		}
	}
	return false
}

// isVideoFile 判断是否为视频文件
func (c *MultimodalAPIClient) isVideoFile(fileType string) bool {
	videoTypes := global.GVA_CONFIG.FileUnderstanding.SupportedFileTypes.Videos
	fileType = strings.ToLower(fileType)
	for _, t := range videoTypes {
		if strings.ToLower(t) == fileType {
			return true
		}
	}
	return false
}

func (c *MultimodalAPIClient) analyzeWithOpenAI(ctx context.Context, filePath, fileType string, modelConfig config.ModelConfig, messageParts []openai.ChatMessagePart) (*example.FileMetadata, error) {
	req := openai.ChatCompletionRequest{
		Model: modelConfig.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:         openai.ChatMessageRoleUser,
				MultiContent: messageParts,
			},
		},
		MaxTokens:   modelConfig.MaxTokens,
		Temperature: modelConfig.Temperature,
	}

	// 设置默认值
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	if req.Model == "" {
		req.Model = openai.GPT4o
	}

	global.GVA_LOG.Info("调用OpenAI API分析文件",
		zap.String("filePath", filePath),
		zap.String("fileType", fileType))

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		global.GVA_LOG.Error("OpenAI API调用失败", zap.Error(err))
		return nil, fmt.Errorf("OpenAI API调用失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI API返回为空")
	}

	content := resp.Choices[0].Message.Content
	global.GVA_LOG.Info("OpenAI API返回结果", zap.String("content", content))

	// 创建 FileMetadata 并设置描述
	metadata := example.FileMetadata{}
	_ = metadata.Set("description", content)
	_ = metadata.Set("contentType", fileType)

	return &metadata, nil
}

func (c *MultimodalAPIClient) getCategoryByFileType(fileType string) string {
	if c.isImageFile(fileType) {
		return "image"
	} else if c.isDocumentFile(fileType) {
		return "document"
	} else if c.isAudioFile(fileType) {
		return "audio"
	} else if c.isVideoFile(fileType) {
		return "video"
	}
	return "unknown"
}

// readFileContent 读取文件内容（支持本地文件和HTTP URL）
func (c *MultimodalAPIClient) readFileContent(filePath string) (string, error) {
	// 判断是否为HTTP URL
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		return c.downloadFileContent(filePath)
	}

	// 读取本地文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := file.Close(); err != nil {
			global.GVA_LOG.Error("关闭文件失败", zap.Error(err))
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// downloadFileContent 下载HTTP URL的文件内容
func (c *MultimodalAPIClient) downloadFileContent(url string) (string, error) {
	global.GVA_LOG.Info("下载文档内容", zap.String("url", url))

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置合适的User-Agent
	req.Header.Set("User-Agent", "Dam-FileUnderstanding/1.0")

	client := &http.Client{
		Timeout: c.timeout, // 使用配置的超时时间
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载文件失败: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			global.GVA_LOG.Error("关闭响应体失败", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应内容失败: %w", err)
	}

	global.GVA_LOG.Info("文档下载完成",
		zap.String("url", url),
		zap.Int("contentLength", len(content)))

	return string(content), nil
}

// downloadAudioFile 下载音频文件到临时位置
func (c *MultimodalAPIClient) downloadAudioFile(ctx context.Context, url string, fileType string) (*os.File, error) {
	global.GVA_LOG.Info("下载音频文件", zap.String("url", url))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置合适的User-Agent
	req.Header.Set("User-Agent", "Dam-FileUnderstanding/1.0")

	client := &http.Client{
		Timeout: c.timeout, // 使用配置的超时时间
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载音频文件失败: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			global.GVA_LOG.Error("关闭响应体失败", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载音频文件失败，状态码: %d", resp.StatusCode)
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp("", fmt.Sprintf("audio_*.%s", fileType))
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}

	// 将响应内容写入临时文件
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			global.GVA_LOG.Error("关闭临时文件失败", zap.Error(closeErr))
		}
		if removeErr := os.Remove(tempFile.Name()); removeErr != nil {
			global.GVA_LOG.Error("删除临时文件失败", zap.Error(removeErr))
		}
		return nil, fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 重置文件指针到开头
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			global.GVA_LOG.Error("关闭临时文件失败", zap.Error(closeErr))
		}
		if removeErr := os.Remove(tempFile.Name()); removeErr != nil {
			global.GVA_LOG.Error("删除临时文件失败", zap.Error(removeErr))
		}
		return nil, fmt.Errorf("重置文件指针失败: %w", err)
	}

	global.GVA_LOG.Info("音频文件下载完成",
		zap.String("url", url),
		zap.String("tempPath", tempFile.Name()))

	return tempFile, nil
}
