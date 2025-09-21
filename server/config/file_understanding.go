package config

// FileUnderstanding 文件理解服务配置
type FileUnderstanding struct {
	APIEndpoint string `mapstructure:"api-endpoint" json:"api-endpoint" yaml:"api-endpoint"` // OpenAI API端点
	APIKey      string `mapstructure:"api-key" json:"api-key" yaml:"api-key"`                // OpenAI API密钥
	Timeout     int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`                // 超时时间（秒）
	Enabled     bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`                // 是否启用文件理解功能

	// 默认配置（向后兼容）
	Model       string  `mapstructure:"model" json:"model" yaml:"model"`                   // 默认模型
	MaxTokens   int     `mapstructure:"max-tokens" json:"max-tokens" yaml:"max-tokens"`    // 默认最大token数
	Temperature float32 `mapstructure:"temperature" json:"temperature" yaml:"temperature"` // 默认温度参数

	// 默认模型配置
	Default ModelConfig `mapstructure:"default" json:"default" yaml:"default"`

	// 按文件类型配置不同模型
	TypeModels TypeModelConfig `mapstructure:"type-models" json:"type-models" yaml:"type-models"`

	// 支持的文件类型配置
	SupportedFileTypes SupportedFileTypes `mapstructure:"supported-file-types" json:"supported-file-types" yaml:"supported-file-types"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	Model       string  `mapstructure:"model" json:"model" yaml:"model"`                   // 使用的模型
	MaxTokens   int     `mapstructure:"max-tokens" json:"max-tokens" yaml:"max-tokens"`    // 最大token数
	Temperature float32 `mapstructure:"temperature" json:"temperature" yaml:"temperature"` // 温度参数
}

// TypeModelConfig 按文件类型的模型配置
type TypeModelConfig struct {
	Image    *ModelConfig `mapstructure:"image" json:"image" yaml:"image"`          // 图片文件模型配置
	Document *ModelConfig `mapstructure:"document" json:"document" yaml:"document"` // 文档文件模型配置
	Video    *ModelConfig `mapstructure:"video" json:"video" yaml:"video"`          // 视频文件模型配置
	Audio    *ModelConfig `mapstructure:"audio" json:"audio" yaml:"audio"`          // 音频文件模型配置
}

// SupportedFileTypes 支持的文件类型配置
type SupportedFileTypes struct {
	Images    []string `mapstructure:"images" json:"images" yaml:"images"`          // 支持的图片格式
	Documents []string `mapstructure:"documents" json:"documents" yaml:"documents"` // 支持的文档格式
	Videos    []string `mapstructure:"videos" json:"videos" yaml:"videos"`          // 支持的视频格式
	Audios    []string `mapstructure:"audios" json:"audios" yaml:"audios"`          // 支持的音频格式
}

// GetModelConfigForFileType 根据文件类型获取模型配置
func (fu *FileUnderstanding) GetModelConfigForFileType(fileType string) ModelConfig {
	// 获取默认配置
	defaultConfig := ModelConfig{
		Model:       fu.Model,
		MaxTokens:   fu.MaxTokens,
		Temperature: fu.Temperature,
	}

	// 如果有新的默认配置，使用新配置
	if fu.Default.Model != "" {
		defaultConfig = fu.Default
	}

	// 根据文件类型返回对应配置
	switch fileType {
	case "image":
		if fu.TypeModels.Image != nil {
			return *fu.TypeModels.Image
		}
	case "document":
		if fu.TypeModels.Document != nil {
			return *fu.TypeModels.Document
		}
	case "video":
		if fu.TypeModels.Video != nil {
			return *fu.TypeModels.Video
		}
	case "audio":
		if fu.TypeModels.Audio != nil {
			return *fu.TypeModels.Audio
		}
	}

	return defaultConfig
}
