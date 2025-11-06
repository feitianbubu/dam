package example

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"sort"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example/request"
	"github.com/flipped-aurora/gin-vue-admin/server/pkg/vectorization"
	"github.com/flipped-aurora/gin-vue-admin/server/service/file_understanding"
	vectorizationService "github.com/flipped-aurora/gin-vue-admin/server/service/vectorization"
	"github.com/flipped-aurora/gin-vue-admin/server/service/vectorization/providers/coze"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/upload"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type FileUploadAndDownloadService struct {
	fileProcessor        file_understanding.FileProcessor
	vectorizationService vectorization.VectorizationService
}

// GetFileProcessor 获取文件处理器，延迟初始化
func (e *FileUploadAndDownloadService) GetFileProcessor() file_understanding.FileProcessor {
	if e.fileProcessor == nil {
		// 创建多模态API客户端
		apiClient := file_understanding.NewMultimodalAPIClient()
		// 创建文件处理器
		e.fileProcessor = file_understanding.NewFileProcessor(apiClient)
	}
	return e.fileProcessor
}

// GetVectorizationService 获取向量化服务，延迟初始化
func (e *FileUploadAndDownloadService) GetVectorizationService() vectorization.VectorizationService {
	if e.vectorizationService == nil {
		// 使用工厂创建向量化服务（默认使用Coze）
		config := &vectorizationService.Config{
			Provider: vectorization.ProviderCoze,
			Settings: global.GVA_CONFIG.Vectorization.Settings,
		}
		service, err := vectorizationService.NewVectorizationService(config)
		if err != nil {
			global.GVA_LOG.Error("创建向量化服务失败", zap.Error(err))
			return nil
		}
		e.vectorizationService = service
	}
	return e.vectorizationService
}

// NewFileUploadAndDownloadService 创建文件上传下载服务
func NewFileUploadAndDownloadService() *FileUploadAndDownloadService {
	// 创建多模态API客户端
	apiClient := file_understanding.NewMultimodalAPIClient()

	// 创建文件处理器
	processor := file_understanding.NewFileProcessor(apiClient)

	return &FileUploadAndDownloadService{
		fileProcessor: processor,
	}
}

func (e *FileUploadAndDownloadService) bizMetadataToMinioMetadata(bizMetadata *example.BizMetadata) map[string]string {
	if bizMetadata == nil {
		return nil
	}

	metadata := make(map[string]string)

	// 处理标签数组 - 转换为JSON字符串（包含中文标签）
	if len(bizMetadata.Tags) > 0 {
		tagsJSON, err := json.Marshal(bizMetadata.Tags)
		if err == nil {
			metadata["tags"] = string(tagsJSON)
			global.GVA_LOG.Debug("将标签存储在MinIO元数据中",
				zap.Strings("标签", bizMetadata.Tags),
				zap.String("JSON", string(tagsJSON)))
		}
	}

	return metadata
}

func (e *FileUploadAndDownloadService) fileToMinioTags(userID uint, userName string, bizMetadata *example.BizMetadata) map[string]string {
	tags := make(map[string]string)

	// MinIO标签只存储用户ID和用户名，自定义标签存储在元数据中
	if userID > 0 {
		tags["user-id"] = fmt.Sprintf("%d", userID)
	}

	if userName != "" {
		tags["username"] = userName
	}

	return tags
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: Upload
//@description: 创建文件上传记录
//@param: file model.ExaFileUploadAndDownload
//@return: error

func (e *FileUploadAndDownloadService) Upload(file *example.ExaFileUploadAndDownload) error {
	return global.GVA_DB.Create(file).Error
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: FindFile
//@description: 查询文件记录
//@param: id uint
//@return: model.ExaFileUploadAndDownload, error

func (e *FileUploadAndDownloadService) FindFile(id uint) (example.ExaFileUploadAndDownload, error) {
	var file example.ExaFileUploadAndDownload
	err := global.GVA_DB.Where("id = ?", id).First(&file).Error
	return file, err
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: DeleteFile
//@description: 删除文件记录
//@param: file model.ExaFileUploadAndDownload
//@return: err error

func (e *FileUploadAndDownloadService) DeleteFile(file example.ExaFileUploadAndDownload) (err error) {
	var fileFromDb example.ExaFileUploadAndDownload
	fileFromDb, err = e.FindFile(file.ID)
	if err != nil {
		return
	}

	// 检查是否还有其他记录引用同一个etag
	//var count int64
	//err = global.GVA_DB.Model(&example.ExaFileUploadAndDownload{}).
	//	Where("etag = ? AND id != ?", fileFromDb.Etag, fileFromDb.ID).
	//	Count(&count).Error
	//if err != nil {
	//	global.GVA_LOG.Error("检查文件引用计数失败",
	//		zap.Uint("fileID", fileFromDb.ID),
	//		zap.String("etag", fileFromDb.Etag),
	//		zap.Error(err))
	//	return errors.New("检查文件引用失败")
	//}
	//
	//// 只有当这是最后一个引用该etag的记录时，才删除OSS文件
	//if count == 0 {
	oss := upload.NewOss()
	if err = oss.DeleteFile(fileFromDb.Key); err != nil {
		global.GVA_LOG.Error("删除OSS文件失败",
			zap.Uint("fileID", fileFromDb.ID),
			zap.String("etag", fileFromDb.Etag),
			zap.Error(err))
		return errors.New("文件删除失败")
	}
	//global.GVA_LOG.Info("OSS文件已删除（最后一个引用）",
	//	zap.Uint("fileID", fileFromDb.ID),
	//	zap.String("etag", fileFromDb.Etag))
	//} else {
	//	global.GVA_LOG.Info("OSS文件保留（仍有其他引用）",
	//		zap.Uint("fileID", fileFromDb.ID),
	//		zap.String("etag", fileFromDb.Etag),
	//		zap.Int64("remainingReferences", count))
	//}

	if fileFromDb.VectorizationDocumentID != "" {
		vectorService := e.GetVectorizationService()
		if vectorService != nil {
			if delErr := vectorService.DeleteDocument(fileFromDb.VectorizationDocumentID); delErr != nil {
				global.GVA_LOG.Error("删除向量化文档失败",
					zap.Uint("fileID", fileFromDb.ID),
					zap.String("documentID", fileFromDb.VectorizationDocumentID),
					zap.Error(delErr))
			}
		}
	}

	err = global.GVA_DB.Where("id = ?", file.ID).Unscoped().Delete(&file).Error
	return err
}

// EditFileName 编辑文件名或者备注
func (e *FileUploadAndDownloadService) EditFileName(file example.ExaFileUploadAndDownload) (err error) {
	var fileFromDb example.ExaFileUploadAndDownload
	return global.GVA_DB.Where("id = ?", file.ID).First(&fileFromDb).Update("name", file.Name).Error
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: GetFileRecordInfoList
//@description: 分页获取数据
//@param: info request.ExaAttachmentCategorySearch
//@return: list interface{}, total int64, err error

func (e *FileUploadAndDownloadService) GetFileRecordInfoList(info request.ExaAttachmentCategorySearch) (list []example.ExaFileUploadAndDownload, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	db := global.GVA_DB.Model(&example.ExaFileUploadAndDownload{})

	if len(info.Keyword) > 0 {
		db = db.Where("name LIKE ?", "%"+info.Keyword+"%")
	}

	if info.ClassId > 0 {
		db = db.Where("class_id = ?", info.ClassId)
	}

	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Limit(limit).Offset(offset).Order("id desc").Find(&list).Error
	return list, total, err
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: UploadFile
//@description: 根据配置文件判断是文件上传到本地或者七牛云
//@param: header *multipart.FileHeader, noSave string
//@return: file model.ExaFileUploadAndDownload, err error

//func (e *FileUploadAndDownloadService) UploadFile(header *multipart.FileHeader, noSave string, classId int) (file example.ExaFileUploadAndDownload, err error) {
//	return e.UploadFileWithMetadata(header, noSave, classId, 0, "", nil, true, false, nil)
//}

func (e *FileUploadAndDownloadService) UploadFileWithMetadata(header *multipart.FileHeader, noSave string, classId int, userID uint, userName string, bizMetadata *example.BizMetadata, autoMetadata bool, waitForMetadata bool, providedMetadata *example.FileMetadata, overwrite bool) (file example.ExaFileUploadAndDownload, err error) {
	// 上传前先检查文件名是否已存在
	var existingFile *example.ExaFileUploadAndDownload
	if noSave == "0" {
		var tempFile example.ExaFileUploadAndDownload
		checkErr := global.GVA_DB.Where("name = ? AND class_id = ?", header.Filename, classId).First(&tempFile).Error
		if checkErr == nil {
			// 文件名已存在
			if overwrite {
				// 允许覆盖，保存现有文件信息，稍后更新而非删除
				existingFile = &tempFile
				global.GVA_LOG.Info("文件名已存在，将覆盖更新",
					zap.String("filename", header.Filename),
					zap.Int("classId", classId),
					zap.Uint("existingFileID", existingFile.ID))

				// 先删除OSS上的旧文件（保留数据库记录）
				if existingFile.Key != "" {
					oss := upload.NewOss()
					if delErr := oss.DeleteFile(existingFile.Key); delErr != nil {
						global.GVA_LOG.Error("删除OSS旧文件失败",
							zap.String("filename", header.Filename),
							zap.String("key", existingFile.Key),
							zap.Error(delErr))
						return file, fmt.Errorf("删除OSS旧文件失败: %w", delErr)
					}
					global.GVA_LOG.Info("OSS旧文件已删除",
						zap.String("filename", header.Filename),
						zap.String("oldKey", existingFile.Key))
				}

				// 如果旧文件有向量化文档ID，删除向量化文档
				if existingFile.VectorizationDocumentID != "" {
					vectorService := e.GetVectorizationService()
					if vectorService != nil {
						if delErr := vectorService.DeleteDocument(existingFile.VectorizationDocumentID); delErr != nil {
							global.GVA_LOG.Error("删除向量化文档失败",
								zap.Uint("fileID", existingFile.ID),
								zap.String("documentID", existingFile.VectorizationDocumentID),
								zap.Error(delErr))
						} else {
							global.GVA_LOG.Info("向量化文档已删除",
								zap.String("documentID", existingFile.VectorizationDocumentID))
						}
					}
				}
			} else {
				// 不允许覆盖，返回错误
				global.GVA_LOG.Warn("文件名已存在，如果需要覆盖请传参数overwrite=true",
					zap.String("filename", header.Filename),
					zap.Int("classId", classId),
					zap.Uint("existingFileID", tempFile.ID))
				return file, errors.New("文件名已存在")
			}
		}
	}

	oss := upload.NewOss()

	minioMetadata := e.bizMetadataToMinioMetadata(bizMetadata)
	minioTags := e.fileToMinioTags(userID, userName, bizMetadata)

	var filePath, key, etag string
	var uploadErr error

	if minioClient, ok := oss.(*upload.Minio); ok && (minioMetadata != nil || minioTags != nil) {
		filePath, key, etag, uploadErr = minioClient.UploadFileWithMetadataAndTags(header, minioMetadata, minioTags)
		global.GVA_LOG.Info("使用MinIO混合方案上传（元数据+标签）",
			zap.String("filename", header.Filename),
			zap.Uint("userID", userID),
			zap.String("userName", userName),
			zap.Any("metadata", minioMetadata),
			zap.Any("tags", minioTags))
	} else if ossWithMetadata, ok := oss.(interface {
		UploadFileWithMetadata(file *multipart.FileHeader, metadata map[string]string) (string, string, string, error)
	}); ok && minioMetadata != nil {
		// 其他支持元数据的存储后端
		filePath, key, etag, uploadErr = ossWithMetadata.UploadFileWithMetadata(header, minioMetadata)
		global.GVA_LOG.Info("使用带元数据的上传方法",
			zap.String("filename", header.Filename),
			zap.Any("metadata", minioMetadata))
	} else {
		// 不支持元数据的存储后端
		filePath, key, uploadErr = oss.UploadFile(header)
		if minioMetadata != nil || minioTags != nil {
			global.GVA_LOG.Warn("存储后端不支持元数据和标签，使用普通上传方法",
				zap.String("filename", header.Filename))
		}
	}
	if uploadErr != nil {
		return file, uploadErr
	}
	s := strings.Split(header.Filename, ".")

	// 构建文件记录
	var f example.ExaFileUploadAndDownload

	// 如果是覆盖模式，保留现有记录的ID和创建时间
	if existingFile != nil {
		f = *existingFile // 复制现有记录，保留ID、CreatedAt等字段
		// 更新新文件信息
		f.Url = filePath
		f.Name = header.Filename
		f.ClassId = classId
		f.FileType = s[len(s)-1]
		f.Key = key
		f.Etag = etag
		f.UserID = userID
		f.Username = userName
		f.ProcessStatus = example.ProcessStatusPending
		f.VectorizationDocumentID = "" // 清空旧的向量化文档ID

		global.GVA_LOG.Info("覆盖模式：保留文件ID和创建时间",
			zap.Uint("fileID", f.ID),
			zap.String("newKey", key),
			zap.String("newEtag", etag))
	} else {
		// 创建新文件记录
		f = example.ExaFileUploadAndDownload{
			Url:           filePath,
			Name:          header.Filename,
			ClassId:       classId,
			FileType:      s[len(s)-1],
			Key:           key,
			Etag:          etag,
			UserID:        userID,
			Username:      userName,
			ProcessStatus: example.ProcessStatusPending,
		}
	}

	// 如果提供了业务元数据，则设置
	if bizMetadata != nil {
		f.BizMetadata = *bizMetadata
	}

	// 如果提供了文件元数据，则直接使用（不再调用文件理解接口）
	if providedMetadata != nil {
		f.Metadata = *providedMetadata
		f.ProcessStatus = example.ProcessStatusCompleted // 标记为已完成
		global.GVA_LOG.Info("使用请求提供的metadata，跳过文件理解",
			zap.String("filename", header.Filename),
			zap.Any("metadata", providedMetadata))
	}

	if noSave == "0" {
		// 根据是否覆盖模式选择不同的保存方式
		if existingFile != nil {
			// 覆盖模式：更新现有记录（GORM的Save会检测到ID存在，执行UPDATE）
			err = global.GVA_DB.Save(&f).Error
			if err != nil {
				return f, err
			}
			global.GVA_LOG.Info("文件覆盖更新成功",
				zap.Uint("fileID", f.ID),
				zap.String("filename", f.Name),
				zap.String("newKey", f.Key))
		} else {
			// 新建模式：创建新记录
			err = e.Upload(&f)
			if err != nil {
				return f, err
			}
		}

		fileID := f.ID
		global.GVA_LOG.Info("文件保存成功", zap.Uint64("fileID", uint64(fileID)))

		// 只有在启用自动生成元数据且未提供metadata的情况下才调用文件理解接口
		if autoMetadata && (providedMetadata == nil) {
			global.GVA_LOG.Info("启用自动生成元数据，准备启动文件理解", zap.Uint64("fileID", uint64(fileID)))

			if waitForMetadata {
				// 同步等待处理完成
				global.GVA_LOG.Info("同步等待文件处理完成", zap.Uint64("fileID", uint64(fileID)))
				e.ProcessFileWithRetry(fileID, 3, "同步文件处理")

				// 重新查询文件获取完整数据（包含metadata）
				f, err = e.FindFile(fileID)
				if err != nil {
					global.GVA_LOG.Error("重新查询文件失败", zap.Uint64("fileID", uint64(fileID)), zap.Error(err))
				}
			} else {
				// 异步处理
				go e.ProcessFileWithRetry(fileID, 3, "异步文件处理")
			}
		} else if !autoMetadata {
			global.GVA_LOG.Info("自动生成元数据已禁用，跳过文件理解", zap.Uint64("fileID", uint64(fileID)))
		}

		return f, nil
	}
	return f, nil
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: ImportURL
//@description: 导入URL
//@param: file model.ExaFileUploadAndDownload
//@return: error

func (e *FileUploadAndDownloadService) ImportURL(file *[]example.ExaFileUploadAndDownload) error {
	return global.GVA_DB.Create(&file).Error
}

// GetFileWithMetadata 获取文件及其元数据
func (e *FileUploadAndDownloadService) GetFileWithMetadata(id uint) (example.ExaFileUploadAndDownload, error) {
	var file example.ExaFileUploadAndDownload
	err := global.GVA_DB.Where("id = ?", id).First(&file).Error
	return file, err
}

// RetryFileProcessing 重试文件处理
func (e *FileUploadAndDownloadService) RetryFileProcessing(id uint) error {
	processor := e.GetFileProcessor()
	return processor.RetryProcessFile(id)
}

// GetFilesByProcessStatus 根据处理状态获取文件列表
func (e *FileUploadAndDownloadService) GetFilesByProcessStatus(status string, pageInfo request.ExaAttachmentCategorySearch) (list []example.ExaFileUploadAndDownload, total int64, err error) {
	limit := pageInfo.PageSize
	offset := pageInfo.PageSize * (pageInfo.Page - 1)
	db := global.GVA_DB.Model(&example.ExaFileUploadAndDownload{}).Where("process_status = ?", status)

	if len(pageInfo.Keyword) > 0 {
		db = db.Where("name LIKE ?", "%"+pageInfo.Keyword+"%")
	}

	if pageInfo.ClassId > 0 {
		db = db.Where("class_id = ?", pageInfo.ClassId)
	}

	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Limit(limit).Offset(offset).Order("id desc").Find(&list).Error
	return list, total, err
}

// searchDocumentIDsByPrompt 通过自然语言提示词搜索文档ID
// searchResult 检索结果结构
type searchResult struct {
	DocumentIDs   []string
	DocumentIDMap map[string]uint  // 文档ID -> 文件ID映射
	Scores        map[uint]float64 // 文件ID -> 分数映射
}

func (e *FileUploadAndDownloadService) searchDocumentIDsWithScores(prompt string, knowledgeIDs []string, topK int, minScore float64, searchType int) (*searchResult, error) {
	// 获取向量化服务
	vectorizationSvc := e.GetVectorizationService()
	if vectorizationSvc == nil {
		return nil, fmt.Errorf("向量化服务不可用")
	}

	// 类型断言为Coze服务
	cozeService, ok := vectorizationSvc.(*coze.CozeService)
	if !ok {
		return nil, fmt.Errorf("当前仅支持Coze向量化服务")
	}

	// 构建检索请求
	retrieveReq := &coze.CozeKnowledgeRetrieveRequest{
		Query:        prompt,
		KnowledgeIDs: knowledgeIDs,
		TopK:         topK,
		MinScore:     minScore,
		SearchType:   searchType,
	}

	// 调用Coze检索API
	resp, err := cozeService.RetrieveKnowledge(retrieveReq)
	if err != nil {
		global.GVA_LOG.Error("Coze知识库检索失败",
			zap.Error(err),
			zap.String("prompt", prompt),
			zap.Strings("knowledgeIDs", knowledgeIDs))
		return nil, fmt.Errorf("知识库检索失败: %w", err)
	}

	// 构建搜索结果，包含分数信息
	result := &searchResult{
		DocumentIDs:   []string{},
		DocumentIDMap: make(map[string]uint),
		Scores:        make(map[uint]float64),
	}

	if len(resp.Results) > 0 {
		documentScores := make(map[string]float64) // document_id -> max_score
		for _, retrieveResult := range resp.Results {
			docID := fmt.Sprintf("%d", retrieveResult.DocumentID)
			if existingScore, exists := documentScores[docID]; !exists || retrieveResult.Score > existingScore {
				documentScores[docID] = retrieveResult.Score
			}
		}

		uniqueDocumentIDs := make([]string, 0, len(documentScores))
		for docID := range documentScores {
			uniqueDocumentIDs = append(uniqueDocumentIDs, docID)
		}

		var files []example.ExaFileUploadAndDownload
		db := global.GVA_DB.Model(&example.ExaFileUploadAndDownload{})
		db = db.Where("vectorization_document_id IN ?", uniqueDocumentIDs)
		err = db.Find(&files).Error
		if err != nil {
			global.GVA_LOG.Warn("查询文件记录映射失败", zap.Error(err))
		}

		for _, dbFile := range files {
			docIDStr := dbFile.VectorizationDocumentID
			result.DocumentIDMap[docIDStr] = dbFile.ID
			result.DocumentIDs = append(result.DocumentIDs, docIDStr)
			result.Scores[dbFile.ID] = documentScores[docIDStr]
		}
	}

	global.GVA_LOG.Info("知识库检索完成",
		zap.String("prompt", prompt),
		zap.Int("totalResults", resp.Total),
		zap.Int("documentCount", len(result.DocumentIDs)),
		zap.Strings("documentIDs", result.DocumentIDs))

	return result, nil
}

// SearchVectorDocuments 执行向量搜索，返回匹配的文档ID和分数
func (e *FileUploadAndDownloadService) SearchVectorDocuments(prompt string, knowledgeIDs []string, topK int, minScore float64, searchType int) (*searchResult, error) {
	// 默认值已在API层的GetSearchDefaults()中设置，这里不再重复处理
	return e.searchDocumentIDsWithScores(prompt, knowledgeIDs, topK, minScore, searchType)
}

func (e *FileUploadAndDownloadService) getDefaultKnowledgeIDs() []string {
	var categories []example.ExaAttachmentCategory
	if err := global.GVA_DB.Where("knowledge_id != ?", "").Find(&categories).Error; err == nil {
		knowledgeIDs := make([]string, 0, len(categories))
		for _, category := range categories {
			if category.KnowledgeID != "" {
				knowledgeIDs = append(knowledgeIDs, category.KnowledgeID)
			}
		}
		return knowledgeIDs
	}
	return []string{}
}

func (e *FileUploadAndDownloadService) GetFileRecordInfoListWithSearch(info request.ExaFileSearchRequest) (list []example.ExaFileUploadAndDownload, total int64, err error) {
	return e.GetFileRecordInfoListWithVectorFilter(info, nil)
}

// GetFileRecordInfoListWithVectorFilter 执行文件搜索，支持向量文档ID过滤
func (e *FileUploadAndDownloadService) GetFileRecordInfoListWithVectorFilter(info request.ExaFileSearchRequest, vectorDocumentIDs []string) (list []example.ExaFileUploadAndDownload, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	db := global.GVA_DB.Model(&example.ExaFileUploadAndDownload{})

	// 传统过滤条件
	if len(info.Keyword) > 0 {
		db = db.Where("name LIKE ?", "%"+info.Keyword+"%")
	}
	if info.ClassId > 0 {
		db = db.Where("class_id = ?", info.ClassId)
	}

	// 业务元数据过滤条件
	e.applyBizMetadataFilters(db, info)

	// Etag过滤
	if info.Etag != "" {
		db = db.Where("etag = ?", info.Etag)
	}

	// 向量文档ID过滤（如果提供）
	if len(vectorDocumentIDs) > 0 {
		db = db.Where("vectorization_document_id IN ?", vectorDocumentIDs)
	}

	// 执行查询
	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Limit(limit).Offset(offset).Order("id desc").Find(&list).Error

	return list, total, err
}

func (e *FileUploadAndDownloadService) ApplyVectorScoresToResults(list []example.ExaFileUploadAndDownload, scores map[uint]float64) []example.ExaFileUploadAndDownload {
	if scores == nil {
		return list
	}

	// 为查询结果填充分数信息
	for i := range list {
		if score, exists := scores[list[i].ID]; exists {
			list[i].Score = score
		}
	}

	// 按分数倒序排序
	sort.Slice(list, func(i, j int) bool {
		return list[i].Score > list[j].Score
	})

	return list
}

// applyBizMetadataFilters 应用业务元数据过滤条件
func (e *FileUploadAndDownloadService) applyBizMetadataFilters(db *gorm.DB, info request.ExaFileSearchRequest) {
	if len(info.Tags) > 0 {
		for _, tag := range info.Tags {
			trimmedTag := strings.TrimSpace(tag)
			if trimmedTag != "" {
				db = db.Where("JSON_CONTAINS(biz_metadata->'$.tags', JSON_QUOTE(?))", trimmedTag)
			}
		}
	}

	// 项目ID过滤
	if info.ProjectId != "" {
		db = db.Where("JSON_EXTRACT(biz_metadata, '$.projectId') = ?", info.ProjectId)
	}

	// 用户ID过滤（优先使用）
	if info.UserId > 0 {
		db = db.Where("user_id = ?", info.UserId)
	}

	// 用户名过滤（可选，用于显示）
	if info.Username != "" {
		db = db.Where("username = ?", info.Username)
	}
}
