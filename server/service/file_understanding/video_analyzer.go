package file_understanding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"go.uber.org/zap"
)

// VideoAnalyzer 视频文件分析器
type VideoAnalyzer struct {
	*BaseAnalyzer
}

// NewVideoAnalyzer 创建视频分析器
func NewVideoAnalyzer(client *MultimodalAPIClient) *VideoAnalyzer {
	return &VideoAnalyzer{
		BaseAnalyzer: NewBaseAnalyzer(client),
	}
}

// SupportedFileTypes 返回支持的视频文件类型
func (a *VideoAnalyzer) SupportedFileTypes() []string {
	return global.GVA_CONFIG.FileUnderstanding.SupportedFileTypes.Videos
}

// IsSupported 检查是否支持指定的文件类型
func (a *VideoAnalyzer) IsSupported(fileType string) bool {
	return a.CheckFileTypeSupport(fileType, a.SupportedFileTypes())
}

// AnalyzeFile 分析视频文件
func (a *VideoAnalyzer) AnalyzeFile(ctx context.Context, filePath string, fileType string, modelConfig config.ModelConfig) (*example.FileMetadata, error) {
	if !a.IsEnabled() {
		a.LogDisabledSkip(filePath, "视频")
		return a.CreateDisabledResponse(filePath, fileType, "视频"), nil
	}

	prompt := `请分析这个视频文件，并以JSON格式返回以下信息：
1. description: 视频内容的详细描述（中文）
2. detectedText: 视频中的文字内容（如字幕、标题等）
3. objects: 检测到的对象、人物、场景元素
4. tags: 视频标签（如：教育、娱乐、新闻、广告等）
5. category: 视频类别
6. language: 检测到的语言（如果有音频）
7. confidence: 分析的置信度（0-1之间的数值）

请确保返回有用的、准确的分析结果。`

	// 检查是否为HTTP URL
	if !strings.HasPrefix(filePath, "http") {
		global.GVA_LOG.Warn("本地视频文件需要转换为HTTP URL才能进行分析",
			zap.String("filePath", filePath))
		return nil, fmt.Errorf("视频文件分析需要HTTP URL，本地文件路径不支持: %s", filePath)
	}

	a.LogAnalysisStart(filePath, fileType, "视频", modelConfig)

	// 由于video_url不是OpenAI SDK标准格式，需要直接构造HTTP请求
	return a.analyzeVideoWithDirectHTTP(ctx, filePath, fileType, prompt, modelConfig)
}

// VideoMessagePart 扩展的消息部分，支持video_url字段
type VideoMessagePart struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	VideoURL map[string]interface{} `json:"video_url,omitempty"`
}

// ExtendedChatCompletionRequest 扩展的ChatCompletionRequest，支持video_url
type ExtendedChatCompletionRequest struct {
	Model       string                          `json:"model"`
	Messages    []ExtendedChatCompletionMessage `json:"messages"`
	MaxTokens   int                             `json:"max_tokens,omitempty"`
	Temperature float32                         `json:"temperature,omitempty"`
}

// ExtendedChatCompletionMessage 扩展的消息，支持video_url内容
type ExtendedChatCompletionMessage struct {
	Role    string             `json:"role"`
	Content []VideoMessagePart `json:"content"`
}

// analyzeVideoWithDirectHTTP 直接使用HTTP请求分析视频文件（支持千问的video_url格式）
func (a *VideoAnalyzer) analyzeVideoWithDirectHTTP(ctx context.Context, filePath string, fileType string, prompt string, modelConfig config.ModelConfig) (*example.FileMetadata, error) {
	fuConfig := global.GVA_CONFIG.FileUnderstanding

	// 构造请求
	req := ExtendedChatCompletionRequest{
		Model:       modelConfig.Model,
		MaxTokens:   modelConfig.MaxTokens,
		Temperature: modelConfig.Temperature,
		Messages: []ExtendedChatCompletionMessage{
			{
				Role: "user",
				Content: []VideoMessagePart{
					{
						Type: "video_url",
						VideoURL: map[string]interface{}{
							"url": filePath,
						},
					},
					{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
	}

	// 设置默认值
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	// 序列化请求体
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	global.GVA_LOG.Info("发送视频分析HTTP请求",
		zap.String("model", modelConfig.Model))

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fuConfig.APIEndpoint+"/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+fuConfig.APIKey)

	// 发送请求
	client := &http.Client{
		Timeout: a.client.timeout,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		global.GVA_LOG.Error("视频分析HTTP请求失败", zap.Error(err))
		return nil, fmt.Errorf("视频分析HTTP请求失败: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			global.GVA_LOG.Error("关闭响应体失败", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		global.GVA_LOG.Error("视频分析请求失败",
			zap.Int("statusCode", resp.StatusCode),
			zap.String("response", string(bodyBytes)))
		return nil, fmt.Errorf("视频分析请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析响应
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("解析API响应失败: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("API返回为空")
	}

	content := apiResponse.Choices[0].Message.Content
	global.GVA_LOG.Info("视频分析完成",
		zap.String("filePath", filePath),
		zap.String("content", content))

	// 构建FileMetadata
	metadata := &example.FileMetadata{
		Description:  content,
		ContentType:  fileType,
		DetectedText: "",
		Objects:      []string{},
		Tags:         []string{"ai-analyzed", "video", fileType},
		FileSize:     0,
		Language:     "auto-detected",
		Category:     "video",
		Confidence:   0.8,
		ExtraData: map[string]interface{}{
			"model":        modelConfig.Model,
			"tokens":       apiResponse.Usage.TotalTokens,
			"rawResponse":  content,
			"analyzed_via": "direct_http",
		},
	}

	return metadata, nil
}
