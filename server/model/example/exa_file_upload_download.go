package example

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

type ExaFileUploadAndDownload struct {
	global.GVA_MODEL
	Name     string `json:"name" form:"name" gorm:"column:name;comment:文件名"`                                // 文件名
	ClassId  int    `json:"classId" form:"classId" gorm:"default:0;type:int;column:class_id;comment:分类id;"` // 分类id
	Url      string `json:"url" form:"url" gorm:"column:url;comment:文件地址"`                                  // 文件地址
	FileType string `json:"fileType" form:"fileType" gorm:"column:file_type;comment:文件类型"`                  // 文件类型
	Key      string `json:"key" form:"key" gorm:"column:key;comment:编号"`                                    // 编号

	// 用户信息 - 一级字段，便于查询和索引
	UserID   uint   `json:"userId" gorm:"column:user_id;index:idx_user_id;comment:上传用户ID"`    // 上传用户ID
	Username string `json:"username" gorm:"column:username;index:idx_username;comment:上传用户名"` // 上传用户名

	// 文件元数据 - 只存储纯文件理解结果
	Metadata FileMetadata `json:"metadata" form:"metadata" gorm:"serializer:json;type:json;column:metadata;comment:文件元数据" swaggertype:"object"`

	// 业务元数据 - 存储业务相关的自定义信息（不包含用户信息）
	BizMetadata BizMetadata `json:"bizMetadata" form:"bizMetadata" gorm:"serializer:json;type:json;column:biz_metadata;comment:业务元数据"`

	// 处理状态相关字段 - 独立管理
	ProcessStatus     string     `json:"processStatus" gorm:"column:process_status;default:pending;comment:处理状态:pending,processing,completed,failed"`
	ProcessedAt       *time.Time `json:"processedAt" gorm:"column:processed_at;comment:处理完成时间"`
	ProcessError      string     `json:"processError" gorm:"column:process_error;comment:处理错误信息"`
	ProcessRetryCount int        `json:"processRetryCount" gorm:"column:process_retry_count;default:0;comment:重试次数"`

	// 向量化服务集成（通用字段）
	VectorizationDocumentID string `json:"vectorizationDocumentId" gorm:"column:vectorization_document_id;comment:向量化服务文档ID"`
	VectorizationStatus     string `json:"vectorizationStatus" gorm:"column:vectorization_status;default:pending;comment:向量化处理状态"`
	VectorizationProvider   string `json:"vectorizationProvider" gorm:"column:vectorization_provider;comment:向量化服务提供商"`
	VectorizationError      string `json:"vectorizationError" gorm:"type:varchar(1000);column:vectorization_error;comment:向量化处理错误信息"`
	VectorizationRetryCount int    `json:"vectorizationRetryCount" gorm:"column:vectorization_retry_count;default:0;comment:向量化重试次数"`

	// 临时字段：向量搜索相似度分数（只在搜索时返回）
	Score float64 `json:"score,omitempty" gorm:"-"` // 向量搜索相似度分数
}

// FileMetadata 只存储文件理解的纯元数据
// 使用 json.RawMessage 提供最大灵活性，支持动态结构
type FileMetadata json.RawMessage

// String 返回 JSON 字符串表示
func (m FileMetadata) String() string {
	if len(m) == 0 {
		return "{}"
	}
	return string(m)
}

// MarshalJSON 实现 json.Marshaler 接口
func (m FileMetadata) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("{}"), nil
	}
	return m, nil
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (m *FileMetadata) UnmarshalJSON(data []byte) error {
	if m == nil {
		return fmt.Errorf("FileMetadata: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

// Set 设置任意键值对
func (m *FileMetadata) Set(key string, value interface{}) error {
	var data map[string]interface{}
	if len(*m) > 0 {
		if err := json.Unmarshal(*m, &data); err != nil {
			return fmt.Errorf("无法解析现有 metadata: %w", err)
		}
	} else {
		data = make(map[string]interface{})
	}

	data[key] = value

	newData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("无法序列化 metadata: %w", err)
	}

	*m = newData
	return nil
}

// Get 获取指定键的值，并解析到目标类型
func (m FileMetadata) Get(key string, target interface{}) error {
	var data map[string]interface{}
	if err := json.Unmarshal(m, &data); err != nil {
		return fmt.Errorf("无法解析 metadata: %w", err)
	}

	value, exists := data[key]
	if !exists {
		return fmt.Errorf("键 %s 不存在", key)
	}

	// 重新序列化后再解析到目标类型（处理类型转换）
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return json.Unmarshal(valueBytes, target)
}

// GetString 获取字符串值
func (m FileMetadata) GetString(key string) (string, error) {
	var result string
	err := m.Get(key, &result)
	return result, err
}

// ToMap 转换为 map 以便访问
func (m FileMetadata) ToMap() (map[string]interface{}, error) {
	var data map[string]interface{}
	if len(m) == 0 {
		return make(map[string]interface{}), nil
	}
	err := json.Unmarshal(m, &data)
	return data, err
}

// FromMap 从 map 创建
func (m *FileMetadata) FromMap(data map[string]interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	*m = bytes
	return nil
}

// IsEmpty 检查是否为空
func (m FileMetadata) IsEmpty() bool {
	return len(m) == 0 || string(m) == "{}" || string(m) == "null"
}

type ImageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// BizMetadata 存储业务相关的自定义信息
// 注意：用户信息（UserID, Username）已提升为 ExaFileUploadAndDownload 的一级字段
type BizMetadata struct {
	// 项目ID - 关联的项目标识
	ProjectID string `json:"projectId" form:"projectId"`

	// 自定义标签 - 用户或业务系统定义的标签
	Tags []string `json:"tags" form:"tags"`

	// 扩展字段 - 其他业务相关的自定义字段
	ExtraFields map[string]interface{} `json:"extraFields" form:"extraFields"`
}

// 处理状态常量
const (
	ProcessStatusPending    = "pending"    // 待处理
	ProcessStatusProcessing = "processing" // 处理中
	ProcessStatusCompleted  = "completed"  // 处理完成
	ProcessStatusFailed     = "failed"     // 处理失败
)

// 向量化状态常量
const (
	VectorizationStatusPending    = "pending"    // 待向量化
	VectorizationStatusProcessing = "processing" // 向量化中
	VectorizationStatusCompleted  = "completed"  // 向量化完成
	VectorizationStatusFailed     = "failed"     // 向量化失败
	VectorizationStatusDisabled   = "disabled"   // 向量化已禁用
)

func (ExaFileUploadAndDownload) TableName() string {
	return "exa_file_upload_and_downloads"
}
