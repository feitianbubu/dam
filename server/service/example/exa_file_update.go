package example

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/pkg/vectorization"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/upload"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (e *FileUploadAndDownloadService) UpdateFile(id uint, opts *example.FileUpdateOptions) (example.ExaFileUploadAndDownload, error) {
	// 查找并验证文件
	file, err := e.findAndValidateFile(id)
	if err != nil {
		return file, err
	}

	// 验证文件名变更
	if err := e.validateFileNameChange(file.Name, opts.Name); err != nil {
		return file, err
	}

	// 如果提供了新文件，先替换 MinIO 中的文件内容
	if opts.NewFile != nil {
		if err := e.replaceFileInStorage(&file, opts.NewFile); err != nil {
			return file, err
		}
		// 文件替换成功后，自动触发重新处理
		opts.Reprocess = true
	}

	// 构建更新字段
	updates, err := e.buildUpdateFields(opts)
	if err != nil {
		return file, err
	}

	// 执行数据库更新
	if len(updates) > 0 {
		if err := global.GVA_DB.Model(&file).Where("id = ?", id).Updates(updates).Error; err != nil {
			return file, err
		}

		// 重新查询更新后的文件信息
		file, err = e.FindFile(id)
		if err != nil {
			return file, err
		}
	}

	// 触发异步任务
	e.triggerAsyncTasks(id, opts.Reprocess, opts.UpdateVector, file.VectorizationDocumentID)

	return file, nil
}

// ========== 辅助函数 ==========

// findAndValidateFile 查找并验证文件是否存在
func (e *FileUploadAndDownloadService) findAndValidateFile(id uint) (example.ExaFileUploadAndDownload, error) {
	var file example.ExaFileUploadAndDownload
	err := global.GVA_DB.Where("id = ?", id).First(&file).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return file, errors.New("文件不存在")
		}
		return file, err
	}
	return file, nil
}

// validateFileNameChange 验证文件名变更是否合法
func (e *FileUploadAndDownloadService) validateFileNameChange(oldName string, newName *string) error {
	// 允许任意修改文件名，包括扩展名
	return nil
}

// buildUpdateFields 构建更新字段（不含文件内容）
func (e *FileUploadAndDownloadService) buildUpdateFields(opts *example.FileUpdateOptions) (map[string]interface{}, error) {
	updates := make(map[string]interface{})

	// 更新基本信息
	if opts.Name != nil {
		updates["name"] = *opts.Name
	}
	if opts.ClassId != nil {
		updates["class_id"] = *opts.ClassId
	}

	// 记录修改人
	if opts.UpdateUserID != nil {
		updates["update_user_id"] = *opts.UpdateUserID
	}

	// 更新���务元数据
	if opts.BizMetadata != nil {
		bizMetadataJSON, err := json.Marshal(opts.BizMetadata)
		if err != nil {
			return nil, err
		}
		updates["biz_metadata"] = string(bizMetadataJSON)
	}

	// 更新文件元数据
	if opts.Metadata != nil {
		updates["metadata"] = string(*opts.Metadata)
	}

	// 重置处理状态
	if opts.Reprocess {
		updates["process_status"] = example.ProcessStatusPending
		updates["process_retry_count"] = 0
		updates["process_error"] = ""
		updates["processed_at"] = nil
	}

	// 重置向量化状态
	if opts.UpdateVector {
		updates["vectorization_status"] = example.VectorizationStatusPending
		updates["vectorization_retry_count"] = 0
		updates["vectorization_error"] = ""
	}

	return updates, nil
}

// triggerAsyncTasks 触发异步任务（文件处理和向量化）
func (e *FileUploadAndDownloadService) triggerAsyncTasks(fileID uint, reprocess, updateVector bool, existingDocID string) {
	// 异步处理文件重新分析
	if reprocess {
		go func() {
			global.GVA_LOG.Info("开始异步重新处理文件", zap.Uint("fileID", fileID))
			processErr := e.ProcessFileWithRetry(fileID, 3, "文件更新后重新处理")
			if processErr != nil {
				global.GVA_LOG.Error("文件重新处理失败",
					zap.Uint("fileID", fileID),
					zap.Error(processErr))
			} else {
				global.GVA_LOG.Info("文件重新处理完成", zap.Uint("fileID", fileID))
			}
		}()
	}

	// 异步更新向量化数据（仅当文档ID存在时）
	if updateVector && existingDocID != "" {
		go func() {
			global.GVA_LOG.Info("开始异步更新向量化数据", zap.Uint("fileID", fileID))
			e.UpdateFileVectorization(fileID)
		}()
	}
}

// getFileExtension 获取文件扩展名
func getFileExtension(filename string) string {
	s := strings.Split(filename, ".")
	if len(s) > 1 {
		return strings.ToLower(s[len(s)-1])
	}
	return ""
}

// UpdateFileVectorization 更新文件向量化数据
func (e *FileUploadAndDownloadService) UpdateFileVectorization(fileID uint) {
	file, err := e.FindFile(fileID)
	if err != nil {
		global.GVA_LOG.Error("获取文件信息失败", zap.Uint("fileID", fileID), zap.Error(err))
		return
	}

	// 检查是否需要向量化
	if file.Metadata.IsEmpty() {
		global.GVA_LOG.Info("文件元数据为空，跳过向量化", zap.Uint("fileID", fileID))
		return
	}

	vectorService := e.GetVectorizationService()
	if vectorService == nil {
		global.GVA_LOG.Error("向量化服务不可用", zap.Uint("fileID", fileID))
		return
	}

	// 更新向量化状态为处理中
	global.GVA_DB.Model(&file).Where("id = ?", fileID).Updates(map[string]interface{}{
		"vectorization_status":      example.VectorizationStatusProcessing,
		"vectorization_retry_count": gorm.Expr("vectorization_retry_count + 1"),
	})

	// 创建上传文档请求
	// 使用默认知识库ID或从配置中获取
	knowledgeBaseID := "default" // 默认知识库ID
	if kbID, ok := global.GVA_CONFIG.Vectorization.Settings["knowledge_base_id"].(string); ok && kbID != "" {
		knowledgeBaseID = kbID
	}

	uploadReq := &vectorization.UploadDocumentRequest{
		KnowledgeBaseID: knowledgeBaseID,
		Name:            file.Name,
		Content:         file.Metadata.String(),
		ContentType:     "application/json",
		Metadata:        file.Metadata,
	}

	// 执行向量化
	document, err := vectorService.UploadDocument(uploadReq)
	if err != nil {
		global.GVA_LOG.Error("向量化处理失败",
			zap.Uint("fileID", fileID),
			zap.Error(err))

		// 更新失败状态
		global.GVA_DB.Model(&file).Where("id = ?", fileID).Updates(map[string]interface{}{
			"vectorization_status": example.VectorizationStatusFailed,
			"vectorization_error":  err.Error(),
		})
		return
	}

	// 更新成功状态
	global.GVA_DB.Model(&file).Where("id = ?", fileID).Updates(map[string]interface{}{
		"vectorization_document_id": document.ID,
		"vectorization_status":      example.VectorizationStatusCompleted,
		"vectorization_error":       "",
	})

	global.GVA_LOG.Info("文件向量化完成",
		zap.Uint("fileID", fileID),
		zap.String("documentID", document.ID))
}

// replaceFileInStorage 替换存储中的文件内容
func (e *FileUploadAndDownloadService) replaceFileInStorage(file *example.ExaFileUploadAndDownload, newFile *multipart.FileHeader) error {
	// 获取 OSS 客户端
	oss := upload.NewOss()

	// 验证是否为 MinIO 客户端
	minioClient, ok := oss.(*upload.Minio)
	if !ok {
		return errors.New("当前仅支持 MinIO 存储方式的文件替换")
	}

	// 使用原有的 key 替换文件内容
	newURL, err := minioClient.ReplaceFile(file.Key, newFile)
	if err != nil {
		global.GVA_LOG.Error("替换文件失败",
			zap.Uint("fileID", file.ID),
			zap.String("key", file.Key),
			zap.Error(err))
		return err
	}

	// 获取新文件类型
	ext := filepath.Ext(newFile.Filename)
	newFileType := strings.TrimPrefix(ext, ".")

	// 更新数据库中的文件信息
	updates := map[string]interface{}{
		"url":       newURL,
		"file_type": newFileType,
	}

	if err := global.GVA_DB.Model(file).Where("id = ?", file.ID).Updates(updates).Error; err != nil {
		global.GVA_LOG.Error("更新文件信息失败",
			zap.Uint("fileID", file.ID),
			zap.Error(err))
		return err
	}

	// 更新内存中的文件对象
	file.Url = newURL
	file.FileType = newFileType

	global.GVA_LOG.Info("文件内容替换成功",
		zap.Uint("fileID", file.ID),
		zap.String("oldType", file.FileType),
		zap.String("newType", newFileType),
		zap.String("key", file.Key))

	return nil
}
