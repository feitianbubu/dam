package config

// Vectorization 向量化服务配置
type Vectorization struct {
	Enable   bool                   `mapstructure:"enable" json:"enable" yaml:"enable"`       // 是否启用向量化服务
	Provider string                 `mapstructure:"provider" json:"provider" yaml:"provider"` // 服务提供商 coze/openai/pinecone/weaviate
	Settings map[string]interface{} `mapstructure:"settings" json:"settings" yaml:"settings"` // 提供商特定配置
	Async    bool                   `mapstructure:"async" json:"async" yaml:"async"`          // 是否异步处理
	Retry    VectorizationRetry     `mapstructure:"retry" json:"retry" yaml:"retry"`          // 重试配置
	Queue    VectorizationQueue     `mapstructure:"queue" json:"queue" yaml:"queue"`          // 队列配置
}

// VectorizationRetry 重试配置
type VectorizationRetry struct {
	Enable      bool `mapstructure:"enable" json:"enable" yaml:"enable"`                   // 是否启用重试
	MaxAttempts int  `mapstructure:"max_attempts" json:"max_attempts" yaml:"max_attempts"` // 最大重试次数
	DelayMs     int  `mapstructure:"delay_ms" json:"delay_ms" yaml:"delay_ms"`             // 重试延迟（毫秒）
	BackoffMs   int  `mapstructure:"backoff_ms" json:"backoff_ms" yaml:"backoff_ms"`       // 退避延迟（毫秒）
}

// VectorizationQueue 队列配置
type VectorizationQueue struct {
	Enable     bool `mapstructure:"enable" json:"enable" yaml:"enable"`                // 是否启用队列
	MaxWorkers int  `mapstructure:"max_workers" json:"max_workers" yaml:"max_workers"` // 最大工作协程数
	BufferSize int  `mapstructure:"buffer_size" json:"buffer_size" yaml:"buffer_size"` // 队列缓冲大小
}

// GetDefaultVectorizationConfig 获取默认向量化配置
func GetDefaultVectorizationConfig() Vectorization {
	return Vectorization{
		Enable:   false,
		Provider: "coze",
		Settings: map[string]interface{}{
			"base_url":           "https://coze.clinx.work",
			"token":              "",
			"space_id":           "",
			"default_dataset_id": "",
			"timeout":            30,
		},
		Async: true,
		Retry: VectorizationRetry{
			Enable:      true,
			MaxAttempts: 3,
			DelayMs:     1000,
			BackoffMs:   2000,
		},
		Queue: VectorizationQueue{
			Enable:     true,
			MaxWorkers: 5,
			BufferSize: 100,
		},
	}
}
