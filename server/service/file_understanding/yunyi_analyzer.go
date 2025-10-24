package file_understanding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
)

// YunyiAnalyzer 云一文件分析器
type YunyiAnalyzer struct {
	*BaseAnalyzer
	apiURL     string
	apiKey     string
	httpClient *http.Client
}

// YunyiRequest 云一API请求结构
type YunyiRequest struct {
	Messages []YunyiMessage `json:"messages"`
}

// YunyiMessage 消息结构
type YunyiMessage struct {
	Content []YunyiContent `json:"content"`
}

// YunyiContent 内容结构
type YunyiContent struct {
	Type string     `json:"type"`
	File *YunyiFile `json:"file,omitempty"`
}

// YunyiFile 文件结构
type YunyiFile struct {
	FileID string `json:"file_id"`
}

// YunyiResponse 云一API响应结构
type YunyiResponse struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []YunyiChoice `json:"choices"`
	Usage   YunyiUsage    `json:"usage"`
}

// YunyiChoice 选择结构
type YunyiChoice struct {
	Index   int `json:"index"`
	Message struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

// YunyiUsage 使用情况结构
type YunyiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewYunyiAnalyzer 创建云一分析器
func NewYunyiAnalyzer(client *MultimodalAPIClient) *YunyiAnalyzer {
	// 从配置中读取云一API设置
	cfg := global.GVA_CONFIG.FileUnderstanding.TypeModels.Yunyi

	return &YunyiAnalyzer{
		BaseAnalyzer: NewBaseAnalyzer(client),
		apiURL:       cfg.APIURL,
		apiKey:       cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (a *YunyiAnalyzer) SetAPIURL(url string) {
	a.apiURL = url
}

func (a *YunyiAnalyzer) SetAPIKey(key string) {
	a.apiKey = key
}

// SupportedFileTypes 返回支持的文件类型
func (a *YunyiAnalyzer) SupportedFileTypes() []string {
	return []string{
		"glb", "gltf", "obj", "fbx", "dae", // 3D模型文件
		"jpg", "jpeg", "png", "gif", "bmp", "webp", // 图片文件
		"mp4", "avi", "mov", "wmv", "flv", "webm", // 视频文件
		"mp3", "wav", "flac", "aac", "ogg", // 音频文件
		"pdf", "doc", "docx", "txt", "rtf", // 文档文件
	}
}

// IsSupported 检查是否支持指定的文件类型
func (a *YunyiAnalyzer) IsSupported(fileType string) bool {
	return a.CheckFileTypeSupport(fileType, a.SupportedFileTypes())
}

// AnalyzeFile 分析文件
func (a *YunyiAnalyzer) AnalyzeFile(ctx context.Context, filePath string, fileType string, modelConfig config.ModelConfig) (*example.FileMetadata, error) {
	if !a.IsEnabled() {
		a.LogDisabledSkip(filePath, "云一")
		return a.CreateDisabledResponse(filePath, fileType, "云一"), nil
	}

	a.LogAnalysisStart(filePath, fileType, "云一", modelConfig)

	// 构建请求
	request := YunyiRequest{
		Messages: []YunyiMessage{
			{
				Content: []YunyiContent{
					{
						Type: "file",
						File: &YunyiFile{
							FileID: filePath,
						},
					},
				},
			},
		},
	}

	// 序列化请求
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建HTTP请求
	uri := fmt.Sprintf("%s/v1/chat/completions", a.apiURL)
	req, err := http.NewRequestWithContext(ctx, "POST", uri, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用云一API失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("云一API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var yunyiResp YunyiResponse
	if err := json.Unmarshal(respBody, &yunyiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应格式
	if len(yunyiResp.Choices) == 0 {
		return nil, fmt.Errorf("云一API返回的响应中没有choices")
	}

	// 将content转换为JSON字符串存储
	contentBytes, err := json.Marshal(yunyiResp.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("转换content失败: %v", err)
	}

	// 创建文件元数据
	metadata := &example.FileMetadata{}
	_ = metadata.Set("description", string(contentBytes))
	_ = metadata.Set("analyzer", "yunyi")
	_ = metadata.Set("api_response", yunyiResp)
	_ = metadata.Set("usage", yunyiResp.Usage)
	_ = metadata.Set("model", yunyiResp.Model)

	// 添加标签
	var tags []string
	tags = append(tags, fileType, "yunyi_analyzed")
	_ = metadata.Set("tags", tags)
	_ = metadata.Set("category", "yunyi")

	// 记录完成日志
	a.LogAnalysisComplete(filePath, "云一", fmt.Sprintf("分析完成，使用token数: %d", yunyiResp.Usage.TotalTokens))

	return metadata, nil
}
