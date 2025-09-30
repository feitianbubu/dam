package file_understanding

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// FileProcessorImpl 文件处理器实现
type FileProcessorImpl struct {
	understandingService     FileUnderstandingService
	vectorizationIntegration *VectorizationIntegration
}

// NewFileProcessor 创建新的文件处理器
func NewFileProcessor(understandingService FileUnderstandingService) FileProcessor {
	// 创建向量化集成服务
	vectorizationIntegration := NewVectorizationIntegration()

	// 初始化向量化服务
	if err := vectorizationIntegration.InitializeVectorizationService(); err != nil {
		global.GVA_LOG.Error("向量化服务初始化失败", zap.Error(err))
	}

	return &FileProcessorImpl{
		understandingService:     understandingService,
		vectorizationIntegration: vectorizationIntegration,
	}
}

// ProcessFile 处理文件，异步调用多模态API并更新数据库
func (p *FileProcessorImpl) ProcessFile(fileID uint) error {
	if err := p.processFileAsync(fileID); err != nil {
		global.GVA_LOG.Error("文件处理失败", zap.Uint64("fileID", uint64(fileID)), zap.Error(err))
	}
	return nil
}

// processFileAsync 异步处理文件
func (p *FileProcessorImpl) processFileAsync(fileID uint) error {
	global.GVA_LOG.Info("开始异步处理文件", zap.Uint64("fileID", uint64(fileID)))

	// 获取文件信息
	var file example.ExaFileUploadAndDownload
	if err := global.GVA_DB.Where("id = ?", fileID).First(&file).Error; err != nil {
		global.GVA_LOG.Error("获取文件信息失败", zap.Uint64("fileID", uint64(fileID)), zap.Error(err))
		return err
	}

	global.GVA_LOG.Info("找到文件，开始处理", zap.Uint64("fileID", uint64(fileID)), zap.String("fileName", file.Name))

	// 更新状态为处理中
	if err := p.UpdateProcessStatus(fileID, example.ProcessStatusProcessing, nil, ""); err != nil {
		return err
	}

	// 调用多模态API分析文件
	metadata, err := p.understandingService.AnalyzeFile(file.Url, file.Tag)
	if err != nil {
		// 处理失败，更新状态
		errMsg := err.Error()
		global.GVA_LOG.Error("文件分析失败", zap.Uint64("fileID", uint64(fileID)), zap.Error(err))
		return p.UpdateProcessStatus(fileID, example.ProcessStatusFailed, nil, errMsg)
	}

	// 处理成功，更新元数据和状态
	now := time.Now()
	file.ProcessedAt = &now
	if err := p.UpdateProcessStatus(fileID, example.ProcessStatusCompleted, metadata, ""); err != nil {
		return err
	}

	// 触发向量化处理
	p.triggerVectorization(fileID)

	return nil
}

// triggerVectorization 触发向量化处理
func (p *FileProcessorImpl) triggerVectorization(fileID uint) {
	if p.vectorizationIntegration != nil && p.vectorizationIntegration.IsEnabled() {
		global.GVA_LOG.Info("开始向量化处理", zap.Uint64("fileID", uint64(fileID)))
		p.vectorizationIntegration.ProcessFileVectorization(fileID)
	} else {
		global.GVA_LOG.Debug("向量化服务未启用或未初始化", zap.Uint64("fileID", uint64(fileID)))
	}
}

// RetryProcessFile 重试处理失败的文件
func (p *FileProcessorImpl) RetryProcessFile(fileID uint) error {
	// 检查文件状态和重试次数
	var file example.ExaFileUploadAndDownload
	if err := global.GVA_DB.Where("id = ?", fileID).First(&file).Error; err != nil {
		return err
	}

	// 检查重试次数限制
	const maxRetryCount = 3
	if file.ProcessRetryCount >= maxRetryCount {
		return errors.New("超过最大重试次数")
	}

	// 增加重试次数
	if err := global.GVA_DB.Model(&file).Update("process_retry_count", file.ProcessRetryCount+1).Error; err != nil {
		return err
	}

	// 重新处理文件
	return p.ProcessFile(fileID)
}

// UpdateProcessStatus 更新文件处理状态
func (p *FileProcessorImpl) UpdateProcessStatus(fileID uint, status string, metadata *example.FileMetadata, errorMsg string) error {
	updates := map[string]interface{}{
		"process_status": status,
		"process_error":  errorMsg,
	}

	if metadata != nil {
		// 将metadata转换为JSON
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			global.GVA_LOG.Error("序列化metadata失败", zap.Uint64("fileID", uint64(fileID)), zap.Error(err))
			return err
		}
		updates["metadata"] = datatypes.JSON(metadataJSON)
	}

	if status == example.ProcessStatusCompleted {
		now := time.Now()
		updates["processed_at"] = &now
	}

	return global.GVA_DB.Model(&example.ExaFileUploadAndDownload{}).
		Where("id = ?", fileID).
		Updates(updates).Error
}
