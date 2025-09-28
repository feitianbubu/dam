package common

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
)

// VectorizationStatusUpdater 向量化状态更新器
type VectorizationStatusUpdater struct{}

// NewVectorizationStatusUpdater 创建向量化状态更新器
func NewVectorizationStatusUpdater() *VectorizationStatusUpdater {
	return &VectorizationStatusUpdater{}
}

// UpdateVectorizationStatus 更新向量化状态
func (u *VectorizationStatusUpdater) UpdateVectorizationStatus(fileID uint, status, errorMsg, provider string) error {
	updates := map[string]interface{}{
		"vectorization_status": status,
		"vectorization_error":  errorMsg,
	}

	if provider != "" {
		updates["vectorization_provider"] = provider
	}

	return global.GVA_DB.Model(&example.ExaFileUploadAndDownload{}).
		Where("id = ?", fileID).
		Updates(updates).Error
}

// UpdateVectorizationStatusToFailed 更新向量化状态为失败
func (u *VectorizationStatusUpdater) UpdateVectorizationStatusToFailed(fileID uint, errorMsg string) error {
	return u.UpdateVectorizationStatus(fileID, example.VectorizationStatusFailed, errorMsg, "")
}

// UpdateVectorizationStatusToCompleted 更新向量化状态为完成
func (u *VectorizationStatusUpdater) UpdateVectorizationStatusToCompleted(fileID uint, provider string) error {
	return u.UpdateVectorizationStatus(fileID, example.VectorizationStatusCompleted, "", provider)
}

// UpdateVectorizationStatusToProcessing 更新向量化状态为处理中
func (u *VectorizationStatusUpdater) UpdateVectorizationStatusToProcessing(fileID uint) error {
	return u.UpdateVectorizationStatus(fileID, example.VectorizationStatusProcessing, "", "")
}

// UpdateVectorizationDocumentID 更新向量化文档ID
func (u *VectorizationStatusUpdater) UpdateVectorizationDocumentID(fileID uint, documentID string) error {
	return global.GVA_DB.Model(&example.ExaFileUploadAndDownload{}).
		Where("id = ?", fileID).
		Update("vectorization_document_id", documentID).Error
}
