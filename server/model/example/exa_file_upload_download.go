package example

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

type ExaFileUploadAndDownload struct {
	global.GVA_MODEL
	Name    string `json:"name" form:"name" gorm:"column:name;comment:文件名"`                                // 文件名
	ClassId int    `json:"classId" form:"classId" gorm:"default:0;type:int;column:class_id;comment:分类id;"` // 分类id
	Url     string `json:"url" form:"url" gorm:"column:url;comment:文件地址"`                                  // 文件地址
	Tag     string `json:"tag" form:"tag" gorm:"column:tag;comment:文件标签"`                                  // 文件标签
	Key     string `json:"key" form:"key" gorm:"column:key;comment:编号"`                                    // 编号

	// 文件元数据 - 只存储纯文件理解结果
	Metadata FileMetadata `json:"metadata" form:"metadata" gorm:"serializer:json;type:json;column:metadata;comment:文件元数据"`

	// 处理状态相关字段 - 独立管理
	ProcessStatus     string     `json:"processStatus" gorm:"column:process_status;default:pending;comment:处理状态:pending,processing,completed,failed"`
	ProcessedAt       *time.Time `json:"processedAt" gorm:"column:processed_at;comment:处理完成时间"`
	ProcessError      string     `json:"processError" gorm:"column:process_error;comment:处理错误信息"`
	ProcessRetryCount int        `json:"processRetryCount" gorm:"column:process_retry_count;default:0;comment:重试次数"`

	// 向量化服务集成（通用字段）
	VectorizationDocumentID string `json:"vectorizationDocumentId" gorm:"column:vectorization_document_id;comment:向量化服务文档ID"`
	VectorizationStatus     string `json:"vectorizationStatus" gorm:"column:vectorization_status;default:pending;comment:向量化处理状态"`
	VectorizationProvider   string `json:"vectorizationProvider" gorm:"column:vectorization_provider;comment:向量化服务提供商"`
	VectorizationError      string `json:"vectorizationError" gorm:"column:vectorization_error;comment:向量化处理错误信息"`
	VectorizationRetryCount int    `json:"vectorizationRetryCount" gorm:"column:vectorization_retry_count;default:0;comment:向量化重试次数"`

	// 临时字段：向量搜索相似度分数（只在搜索时返回）
	Score float64 `json:"score,omitempty" gorm:"-"` // 向量搜索相似度分数
}

// FileMetadata 只存储文件理解的纯元数据
type FileMetadata struct {
	Description  string                 `json:"description"`  // AI生成的文件描述
	ContentType  string                 `json:"contentType"`  // 检测到的内容类型
	DetectedText string                 `json:"detectedText"` // 提取的文本内容
	Objects      []string               `json:"objects"`      // 检测到的对象/实体
	Tags         []string               `json:"tags"`         // 自动生成的标签
	FileSize     int64                  `json:"fileSize"`     // 文件大小
	Dimensions   *ImageDimensions       `json:"dimensions"`   // 图片尺寸(如果是图片)
	Language     string                 `json:"language"`     // 检测到的语言
	Category     string                 `json:"category"`     // 文件类别
	Confidence   float64                `json:"confidence"`   // AI理解的置信度
	ExtraData    map[string]interface{} `json:"extraData"`    // 扩展数据
}

type ImageDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
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
