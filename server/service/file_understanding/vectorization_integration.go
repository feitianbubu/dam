package file_understanding

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/pkg/vectorization"
	vectorizationService "github.com/flipped-aurora/gin-vue-admin/server/service/vectorization"
	"go.uber.org/zap"
)

// VectorizationIntegration 向量化集成服务
type VectorizationIntegration struct {
	businessService *vectorizationService.BusinessService
}

// NewVectorizationIntegration 创建向量化集成服务
func NewVectorizationIntegration() *VectorizationIntegration {
	return &VectorizationIntegration{}
}

// InitializeVectorizationService 初始化向量化服务
func (v *VectorizationIntegration) InitializeVectorizationService() error {
	// 检查是否启用向量化
	if !global.GVA_CONFIG.Vectorization.Enable {
		global.GVA_LOG.Info("向量化服务已禁用")
		return nil
	}

	// 创建向量化配置
	config := &vectorizationService.Config{
		Provider: vectorization.Provider(global.GVA_CONFIG.Vectorization.Provider),
		Settings: global.GVA_CONFIG.Vectorization.Settings,
	}

	// 验证配置
	if err := vectorizationService.ValidateConfig(config); err != nil {
		global.GVA_LOG.Error("向量化服务配置验证失败", zap.Error(err))
		return err
	}

	// 创建向量化服务
	vectorService, err := vectorizationService.NewVectorizationService(config)
	if err != nil {
		global.GVA_LOG.Error("创建向量化服务失败", zap.Error(err))
		return err
	}

	// 创建业务服务
	v.businessService = vectorizationService.NewBusinessService(vectorService, config)

	global.GVA_LOG.Info("向量化服务初始化成功",
		zap.String("provider", global.GVA_CONFIG.Vectorization.Provider))

	return nil
}

// ProcessFileVectorization 处理文件向量化
func (v *VectorizationIntegration) ProcessFileVectorization(fileID uint) {
	if v.businessService == nil {
		global.GVA_LOG.Debug("向量化服务未初始化", zap.Uint64("fileID", uint64(fileID)))
		return
	}

	if err := v.businessService.ProcessFileVectorization(fileID); err != nil {
		global.GVA_LOG.Error("文件向量化处理失败",
			zap.Uint64("fileID", uint64(fileID)),
			zap.Error(err))

		// 更新向量化状态为失败
		v.updateVectorizationStatusToFailed(fileID, err.Error())
	}
}

// updateVectorizationStatusToFailed 更新向量化状态为失败
func (v *VectorizationIntegration) updateVectorizationStatusToFailed(fileID uint, errorMsg string) {
	updates := map[string]interface{}{
		"vectorization_status": example.VectorizationStatusFailed,
		"vectorization_error":  errorMsg,
	}

	if err := global.GVA_DB.Model(&example.ExaFileUploadAndDownload{}).
		Where("id = ?", fileID).
		Updates(updates).Error; err != nil {
		global.GVA_LOG.Error("更新向量化状态失败",
			zap.Uint64("fileID", uint64(fileID)),
			zap.Error(err))
	}
}

// GetBusinessService 获取业务服务（用于API调用）
func (v *VectorizationIntegration) GetBusinessService() *vectorizationService.BusinessService {
	return v.businessService
}

// IsEnabled 检查向量化服务是否启用
func (v *VectorizationIntegration) IsEnabled() bool {
	return global.GVA_CONFIG.Vectorization.Enable && v.businessService != nil
}
