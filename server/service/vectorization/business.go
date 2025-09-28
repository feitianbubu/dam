package vectorization

import (
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/pkg/vectorization"
	"github.com/flipped-aurora/gin-vue-admin/server/service/common"
	"go.uber.org/zap"
)

// BusinessService 向量化业务服务
type BusinessService struct {
	vectorService vectorization.VectorizationService
	config        *Config
	statusUpdater *common.VectorizationStatusUpdater
}

// NewBusinessService 创建向量化业务服务
func NewBusinessService(vectorService vectorization.VectorizationService, config *Config) *BusinessService {
	return &BusinessService{
		vectorService: vectorService,
		config:        config,
		statusUpdater: common.NewVectorizationStatusUpdater(),
	}
}

// ProcessFileVectorization 处理文件向量化
func (s *BusinessService) ProcessFileVectorization(fileID uint) error {
	// 检查是否启用向量化
	if !global.GVA_CONFIG.Vectorization.Enable {
		global.GVA_LOG.Debug("向量化服务已禁用", zap.Uint64("fileID", uint64(fileID)))
		return nil
	}

	// 获取文件信息
	var file example.ExaFileUploadAndDownload
	if err := global.GVA_DB.Where("id = ?", fileID).First(&file).Error; err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 检查文件是否已经完成处理
	if file.ProcessStatus != example.ProcessStatusCompleted {
		return fmt.Errorf("文件尚未完成处理，无法进行向量化")
	}

	// 检查是否已经向量化过
	if file.VectorizationStatus == example.VectorizationStatusCompleted {
		global.GVA_LOG.Info("文件已完成向量化", zap.Uint64("fileID", uint64(fileID)))
		return nil
	}

	// 更新向量化状态为处理中
	if err := s.statusUpdater.UpdateVectorizationStatusToProcessing(fileID); err != nil {
		return fmt.Errorf("更新向量化状态失败: %w", err)
	}

	// 根据配置决定是否异步处理
	if global.GVA_CONFIG.Vectorization.Async {
		go s.processFileVectorizationAsync(fileID, &file)
		return nil
	}

	return s.processFileVectorizationSync(fileID, &file)
}

// processFileVectorizationAsync 异步处理文件向量化
func (s *BusinessService) processFileVectorizationAsync(fileID uint, file *example.ExaFileUploadAndDownload) {
	if err := s.processFileVectorizationSync(fileID, file); err != nil {
		global.GVA_LOG.Error("异步向量化处理失败",
			zap.Uint64("fileID", uint64(fileID)),
			zap.Error(err))
	}
}

// processFileVectorizationSync 同步处理文件向量化
func (s *BusinessService) processFileVectorizationSync(fileID uint, file *example.ExaFileUploadAndDownload) error {
	var err error

	// 重试机制
	maxAttempts := 1
	if global.GVA_CONFIG.Vectorization.Retry.Enable {
		maxAttempts = global.GVA_CONFIG.Vectorization.Retry.MaxAttempts
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = s.doVectorization(fileID, file)
		if err == nil {
			// 成功，更新状态
			return s.statusUpdater.UpdateVectorizationStatusToCompleted(fileID, global.GVA_CONFIG.Vectorization.Provider)
		}

		global.GVA_LOG.Warn("向量化处理失败，尝试重试",
			zap.Uint64("fileID", uint64(fileID)),
			zap.Int("attempt", attempt),
			zap.Int("maxAttempts", maxAttempts),
			zap.Error(err))

		// 如果不是最后一次尝试，等待一段时间后重试
		if attempt < maxAttempts && global.GVA_CONFIG.Vectorization.Retry.Enable {
			delay := time.Duration(global.GVA_CONFIG.Vectorization.Retry.DelayMs) * time.Millisecond
			backoff := time.Duration(global.GVA_CONFIG.Vectorization.Retry.BackoffMs) * time.Millisecond
			time.Sleep(delay + time.Duration(attempt-1)*backoff)
		}
	}

	// 所有重试都失败，更新状态
	s.statusUpdater.UpdateVectorizationStatusToFailed(fileID, err.Error())
	return err
}

// doVectorization 执行向量化操作
func (s *BusinessService) doVectorization(fileID uint, file *example.ExaFileUploadAndDownload) error {
	// 构建文档内容
	content := s.buildDocumentContent(file)

	// 获取默认知识库ID
	defaultDatasetID, ok := global.GVA_CONFIG.Vectorization.Settings["default_dataset_id"].(string)
	if !ok || defaultDatasetID == "" {
		return fmt.Errorf("未配置默认知识库ID")
	}

	// 构建上传请求
	req := &vectorization.UploadDocumentRequest{
		KnowledgeBaseID: defaultDatasetID,
		Name:            file.Name,
		Content:         content,
		ContentType:     file.Metadata.ContentType,
		Metadata: map[string]string{
			"file_id":     fmt.Sprintf("%d", file.ID),
			"file_size":   fmt.Sprintf("%d", file.Metadata.FileSize),
			"language":    file.Metadata.Language,
			"category":    file.Metadata.Category,
			"confidence":  fmt.Sprintf("%.2f", file.Metadata.Confidence),
			"upload_time": time.Now().Format(time.RFC3339),
		},
		ChunkStrategy:   GetDefaultChunkStrategy(),
		ParsingStrategy: GetDefaultParsingStrategy(),
	}

	// 上传文档
	document, err := s.vectorService.UploadDocument(req)
	if err != nil {
		return fmt.Errorf("上传文档到向量化服务失败: %w", err)
	}

	// 更新数据库记录文档ID
	if err := s.statusUpdater.UpdateVectorizationDocumentID(fileID, document.ID); err != nil {
		return fmt.Errorf("更新向量化文档ID失败: %w", err)
	}

	global.GVA_LOG.Info("文档向量化上传成功",
		zap.Uint64("fileID", uint64(fileID)),
		zap.String("documentID", document.ID),
		zap.String("provider", global.GVA_CONFIG.Vectorization.Provider))

	return nil
}

// buildDocumentContent 构建文档内容
func (s *BusinessService) buildDocumentContent(file *example.ExaFileUploadAndDownload) string {
	var content strings.Builder

	// 文件基本信息
	content.WriteString(fmt.Sprintf("# %s\n\n", file.Name))
	content.WriteString(fmt.Sprintf("**文件类型**: %s\n", file.Metadata.ContentType))
	content.WriteString(fmt.Sprintf("**文件大小**: %d bytes\n", file.Metadata.FileSize))

	if file.Metadata.Language != "" {
		content.WriteString(fmt.Sprintf("**语言**: %s\n", file.Metadata.Language))
	}

	if file.Metadata.Category != "" {
		content.WriteString(fmt.Sprintf("**分类**: %s\n", file.Metadata.Category))
	}

	content.WriteString("\n")

	// AI生成的描述
	if file.Metadata.Description != "" {
		content.WriteString(fmt.Sprintf("## 内容描述\n%s\n\n", file.Metadata.Description))
	}

	// 提取的文本内容
	if file.Metadata.DetectedText != "" {
		content.WriteString(fmt.Sprintf("## 文本内容\n%s\n\n", file.Metadata.DetectedText))
	}

	// 标签
	if len(file.Metadata.Tags) > 0 {
		content.WriteString(fmt.Sprintf("## 标签\n%s\n\n", strings.Join(file.Metadata.Tags, ", ")))
	}

	// 检测到的对象
	if len(file.Metadata.Objects) > 0 {
		content.WriteString(fmt.Sprintf("## 检测对象\n%s\n\n", strings.Join(file.Metadata.Objects, ", ")))
	}

	// 图片尺寸信息
	if file.Metadata.Dimensions != nil {
		content.WriteString(fmt.Sprintf("## 图片信息\n宽度: %d像素\n高度: %d像素\n\n",
			file.Metadata.Dimensions.Width, file.Metadata.Dimensions.Height))
	}

	// 置信度信息
	if file.Metadata.Confidence > 0 {
		content.WriteString(fmt.Sprintf("## AI分析置信度\n%.2f%%\n\n", file.Metadata.Confidence*100))
	}

	return content.String()
}

// updateVectorizationStatus 更新向量化状态（已废弃，请使用statusUpdater）
// Deprecated: 使用statusUpdater.UpdateVectorizationStatus替代
func (s *BusinessService) updateVectorizationStatus(fileID uint, status, errorMsg, provider string) error {
	return s.statusUpdater.UpdateVectorizationStatus(fileID, status, errorMsg, provider)
}

// GetVectorizationProgress 获取向量化进度
func (s *BusinessService) GetVectorizationProgress(fileID uint) (*vectorization.DocumentProgress, error) {
	var file example.ExaFileUploadAndDownload
	if err := global.GVA_DB.Where("id = ?", fileID).First(&file).Error; err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	if file.VectorizationDocumentID == "" {
		return &vectorization.DocumentProgress{
			DocumentID:     "",
			Progress:       0,
			Status:         file.VectorizationStatus,
			DocumentName:   file.Name,
			StatusDescript: "尚未开始向量化",
		}, nil
	}

	// 从向量化服务获取进度
	return s.vectorService.GetDocumentProgress(file.VectorizationDocumentID)
}

// RetryVectorization 重试向量化
func (s *BusinessService) RetryVectorization(fileID uint) error {
	// 重置向量化状态
	if err := s.statusUpdater.UpdateVectorizationStatus(fileID, example.VectorizationStatusPending, "", ""); err != nil {
		return fmt.Errorf("重置向量化状态失败: %w", err)
	}

	// 重新处理
	return s.ProcessFileVectorization(fileID)
}

// SearchSimilarFiles 搜索相似文件（基于向量化）
func (s *BusinessService) SearchSimilarFiles(query string, topK int) (*vectorization.SearchResult, error) {
	defaultDatasetID, ok := global.GVA_CONFIG.Vectorization.Settings["default_dataset_id"].(string)
	if !ok || defaultDatasetID == "" {
		return nil, fmt.Errorf("未配置默认知识库ID")
	}

	req := &vectorization.SearchRequest{
		Query:           query,
		KnowledgeBaseID: defaultDatasetID,
		TopK:            topK,
	}

	return s.vectorService.SearchDocuments(req)
}
