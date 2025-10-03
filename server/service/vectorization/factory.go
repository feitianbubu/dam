package vectorization

import (
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/pkg/vectorization"
	"github.com/flipped-aurora/gin-vue-admin/server/service/vectorization/providers/coze"
)

// Config 向量化服务配置
type Config struct {
	Provider vectorization.Provider `mapstructure:"provider" json:"provider" yaml:"provider"`
	Settings map[string]interface{} `mapstructure:"settings" json:"settings" yaml:"settings"`
}

// NewVectorizationService 创建向量化服务实例
func NewVectorizationService(config *Config) (vectorization.VectorizationService, error) {
	if config == nil {
		return nil, fmt.Errorf("vectorization config is required")
	}

	switch config.Provider {
	case vectorization.ProviderCoze:
		return coze.NewCozeService(config.Settings)
	case vectorization.ProviderOpenAI:
		return nil, fmt.Errorf("openai provider not implemented yet")
	case vectorization.ProviderPinecone:
		return nil, fmt.Errorf("pinecone provider not implemented yet")
	case vectorization.ProviderWeaviate:
		return nil, fmt.Errorf("weaviate provider not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported vectorization provider: %s", config.Provider)
	}
}

// NewVectorizationServiceFromGlobalConfig 从全局配置创建向量化服务实例
func NewVectorizationServiceFromGlobalConfig() (vectorization.VectorizationService, error) {
	if !global.GVA_CONFIG.Vectorization.Enable {
		return nil, fmt.Errorf("vectorization service is disabled")
	}

	config := &Config{
		Provider: vectorization.Provider(global.GVA_CONFIG.Vectorization.Provider),
		Settings: global.GVA_CONFIG.Vectorization.Settings,
	}

	return NewVectorizationService(config)
}

// ValidateConfig 验证配置
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	if config.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	switch config.Provider {
	case vectorization.ProviderCoze:
		return validateCozeConfig(config.Settings)
	case vectorization.ProviderOpenAI:
		return fmt.Errorf("openai provider not implemented yet")
	case vectorization.ProviderPinecone:
		return fmt.Errorf("pinecone provider not implemented yet")
	case vectorization.ProviderWeaviate:
		return fmt.Errorf("weaviate provider not implemented yet")
	default:
		return fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// validateCozeConfig 验证Coze配置
func validateCozeConfig(settings map[string]interface{}) error {
	requiredFields := []string{"base_url", "token"}

	for _, field := range requiredFields {
		if value, ok := settings[field]; !ok || value == "" {
			return fmt.Errorf("coze config missing required field: %s", field)
		}
	}

	return nil
}

// GetDefaultChunkStrategy 获取默认分块策略
func GetDefaultChunkStrategy() *vectorization.ChunkStrategy {
	return &vectorization.ChunkStrategy{
		Separator:         ".",
		MaxTokens:         1000,
		RemoveExtraSpaces: true,
		ChunkType:         vectorization.ChunkTypeDefault,
		Overlap:           100,
	}
}

// GetDefaultParsingStrategy 获取默认解析策略
func GetDefaultParsingStrategy() *vectorization.ParsingStrategy {
	return &vectorization.ParsingStrategy{
		ParsingType:     1,
		ImageExtraction: true,
		TableExtraction: true,
		ImageOCR:        false,
	}
}
