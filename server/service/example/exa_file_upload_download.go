package example

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"sort"
	"strings"
	"time"

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

func (e *FileUploadAndDownloadService) DeleteFile(file example.ExaFileUploadAndDownload, currentUserID uint, currentUserAuthorityID uint) (err error) {
	// 检查权限
	fileFromDb, err := e.checkFileOwnership(file.ID, currentUserID, currentUserAuthorityID)
	if err != nil {
		return err
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

	// 使用统一的辅助函数删除OSS文件和向量化文档
	if err = e.deleteOldFileAndVectorization(fileFromDb); err != nil {
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

	err = global.GVA_DB.Where("id = ?", file.ID).Unscoped().Delete(&file).Error
	return err
}

// deleteOSSFile 删除OSS文件
func (e *FileUploadAndDownloadService) deleteOSSFile(file *example.ExaFileUploadAndDownload) error {
	if file.Key == "" {
		return nil
	}

	oss := upload.NewOss()
	if err := oss.DeleteFile(file.Key); err != nil {
		global.GVA_LOG.Error("删除OSS文件失败",
			zap.Uint("fileID", file.ID),
			zap.String("filename", file.Name),
			zap.String("key", file.Key),
			zap.Error(err))
		return err
	}

	global.GVA_LOG.Info("OSS文件已删除",
		zap.Uint("fileID", file.ID),
		zap.String("filename", file.Name),
		zap.String("key", file.Key))
	return nil
}

// deleteVectorizationDocument 删除向量化文档
func (e *FileUploadAndDownloadService) deleteVectorizationDocument(file *example.ExaFileUploadAndDownload) error {
	if file.VectorizationDocumentID == "" {
		return nil
	}

	vectorService := e.GetVectorizationService()
	if vectorService == nil {
		global.GVA_LOG.Warn("向量化服务不可用，跳过删除向量化文档",
			zap.Uint("fileID", file.ID),
			zap.String("documentID", file.VectorizationDocumentID))
		return nil
	}

	if err := vectorService.DeleteDocument(file.VectorizationDocumentID); err != nil {
		global.GVA_LOG.Error("删除向量化文档失败",
			zap.Uint("fileID", file.ID),
			zap.String("documentID", file.VectorizationDocumentID),
			zap.Error(err))
		// 向量化文档删除失败不阻断流程，仅记录日志
		return nil
	}

	global.GVA_LOG.Info("向量化文档已删除",
		zap.Uint("fileID", file.ID),
		zap.String("documentID", file.VectorizationDocumentID))
	return nil
}

// deleteOldFileAndVectorization 删除OSS文件和向量化文档（组合函数）
func (e *FileUploadAndDownloadService) deleteOldFileAndVectorization(file *example.ExaFileUploadAndDownload) error {
	// 先删除OSS文件
	if err := e.deleteOSSFile(file); err != nil {
		return err
	}

	// 再删除向量化文档（不阻断流程）
	err := e.deleteVectorizationDocument(file)
	if err != nil {
		global.GVA_LOG.Warn("deleteVectorizationDocument fail in deleteOldFileAndVectorization", zap.Error(err))
	}

	return nil
}

// isFileAdmin 判断用户是否是文件管理员
// 管理员角色ID为888（admin用户的默认角色）
func (e *FileUploadAndDownloadService) isFileAdmin(authorityID uint) bool {
	// 可以在这里扩展更多的管理员角色ID
	// 例如：return authorityID == 888 || authorityID == 999
	return authorityID == 888
}

// checkFileOwnership 检查用户是否有权限操作文件
// 只有文件所有者或管理员才能操作文件
func (e *FileUploadAndDownloadService) checkFileOwnership(fileID uint, currentUserID uint, currentUserAuthorityID uint) (*example.ExaFileUploadAndDownload, error) {
	// 查询文件
	var file example.ExaFileUploadAndDownload
	err := global.GVA_DB.Where("id = ?", fileID).First(&file).Error
	if err != nil {
		return nil, errors.New("文件不存在")
	}

	// 管理员可以操作所有文件
	if e.isFileAdmin(currentUserAuthorityID) {
		return &file, nil
	}

	// 普通用户只能操作自己上传的文件（检查UserID，而不是UpdateUserID）
	if file.UserID != currentUserID {
		return nil, errors.New("无权操作该文件，只能操作自己上传的文件")
	}

	return &file, nil
}

// EditFileName 编辑文件名或者备注
func (e *FileUploadAndDownloadService) EditFileName(file example.ExaFileUploadAndDownload, currentUserID uint, currentUserAuthorityID uint) (err error) {
	// 检查权限
	_, err = e.checkFileOwnership(file.ID, currentUserID, currentUserAuthorityID)
	if err != nil {
		return err
	}

	// 执行更新
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
//@function: UploadFileWithMetadata
//@description: 根据配置文件判断是文件上传到本地或者七牛云
//@param: opts *example.FileUploadOptions
//@return: file model.ExaFileUploadAndDownload, err error

func (e *FileUploadAndDownloadService) UploadFileWithMetadata(opts *example.FileUploadOptions) (file example.ExaFileUploadAndDownload, err error) {
	// 参数验证
	if opts == nil || opts.FileHeader == nil {
		return file, errors.New("上传参数不能为空")
	}

	header := opts.FileHeader
	noSave := opts.NoSave
	classId := opts.ClassId
	userID := opts.UserID
	userName := opts.UserName
	authorityID := opts.AuthorityID
	bizMetadata := opts.BizMetadata
	autoMetadata := opts.AutoMetadata
	waitForMetadata := opts.WaitForMetadata
	providedMetadata := opts.ProvidedMetadata
	overwrite := opts.Overwrite
	publicRead := opts.PublicRead

	// 上传前先检查文件名是否已存在
	if noSave == "0" {
		var existingFile example.ExaFileUploadAndDownload
		checkErr := global.GVA_DB.Where("name = ? AND class_id = ?", header.Filename, classId).First(&existingFile).Error
		if checkErr == nil {
			if overwrite {
				global.GVA_LOG.Info("文件名已存在，直接调用UpdateFile接口覆盖",
					zap.String("filename", header.Filename),
					zap.Int("classId", classId),
					zap.Uint("existingFileID", existingFile.ID))

				updateOpts := &example.FileUpdateOptions{
					NewFile:      header,
					BizMetadata:  bizMetadata,
					Reprocess:    autoMetadata,
					UpdateVector: false,
					UpdateUserID: &userID,
				}

				if providedMetadata != nil {
					updateOpts.Metadata = providedMetadata
				}

				// 只在用户明确传了 publicRead 参数时才设置（nil表示不修改）
				if publicRead != nil {
					updateOpts.PublicRead = publicRead
				}

				return e.UpdateFile(existingFile.ID, updateOpts, userID, authorityID)
			} else {
				err = errors.New("文件名已存在，如果需要覆盖请传参数overwrite=true")
				return file, err
			}
		}
	}

	oss := upload.NewOss()

	minioMetadata := e.bizMetadataToMinioMetadata(bizMetadata)
	minioTags := e.fileToMinioTags(userID, userName, bizMetadata)

	var filePath, key, etag string
	var uploadErr error

	// 确定实际的 publicRead 值（默认为 false）
	actualPublicRead := false
	if publicRead != nil {
		actualPublicRead = *publicRead
	}

	// 优先检查是否支持 ACL
	if ossWithACL, ok := oss.(upload.OSSWithACL); ok {
		// 支持 ACL 的存储后端（如 AWS S3）
		filePath, key, etag, uploadErr = ossWithACL.UploadFileWithACL(header, minioMetadata, actualPublicRead)
		global.GVA_LOG.Info("使用支持ACL的上传方法",
			zap.String("filename", header.Filename),
			zap.Bool("publicRead", actualPublicRead),
			zap.Any("metadata", minioMetadata))
	} else if publicRead != nil && *publicRead {
		// 不支持 ACL 但用户明确要求公开读，返回错误
		return file, errors.New("当前存储后端不支持公开读（ACL）设置，请联系管理员配置支持ACL的存储服务")
	} else if minioClient, ok := oss.(*upload.Minio); ok && (minioMetadata != nil || minioTags != nil) {
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

	// 构建文件记录（覆盖模式已在前面通过UpdateFile处理，这里只处理新建）
	f := example.ExaFileUploadAndDownload{
		Url:           filePath,
		Name:          header.Filename,
		ClassId:       classId,
		FileType:      s[len(s)-1],
		Key:           key,
		Etag:          etag,
		UserID:        userID,
		Username:      userName,
		PublicRead:    actualPublicRead,
		ProcessStatus: example.ProcessStatusPending,
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
		err = e.Upload(&f)
		if err != nil {
			return f, err
		}

		fileID := f.ID
		global.GVA_LOG.Info("文件保存成功", zap.Uint64("fileID", uint64(fileID)))

		if autoMetadata && (providedMetadata == nil) {
			global.GVA_LOG.Info("启用自动生成元数据，准备启动文件理解", zap.Uint64("fileID", uint64(fileID)))

			if waitForMetadata {
				global.GVA_LOG.Info("同步等待文件处理完成", zap.Uint64("fileID", uint64(fileID)))
				e.ProcessFileWithRetry(fileID, 3, "同步文件处理")

				f, err = e.FindFile(fileID)
				if err != nil {
					global.GVA_LOG.Error("重新查询文件失败", zap.Uint64("fileID", uint64(fileID)), zap.Error(err))
				}
			} else {
				go e.ProcessFileWithRetry(fileID, 3, "异步文件处理")
			}
		} else if !autoMetadata {
			global.GVA_LOG.Info("自动生成元数据已禁用，跳过文件理解", zap.Uint64("fileID", uint64(fileID)))
		}
	}
	f.Url = e.GetPresignedURL(&f, time.Hour)
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

//func (e *FileUploadAndDownloadService) GetFileRecordInfoListWithSearch(info request.ExaFileSearchRequest) (list []example.ExaFileUploadAndDownload, total int64, err error) {
//	return e.GetFileRecordInfoListWithVectorFilter(info, nil)
//}

func (e *FileUploadAndDownloadService) GetFileRecordInfoListWithVectorFilter(info request.ExaFileSearchRequest, vectorDocumentIDs []string) (list []example.ExaFileUploadAndDownload, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	db := global.GVA_DB.Model(&example.ExaFileUploadAndDownload{})

	// 关键字或向量文档ID过滤（OR关系）
	if len(info.Keyword) > 0 && len(vectorDocumentIDs) > 0 {
		db = db.Where(
			global.GVA_DB.Where("name LIKE ?", "%"+info.Keyword+"%").Or("vectorization_document_id IN ?", vectorDocumentIDs),
		)
	} else if len(info.Keyword) > 0 {
		db = db.Where("name LIKE ?", "%"+info.Keyword+"%")
	} else if len(vectorDocumentIDs) > 0 {
		db = db.Where("vectorization_document_id IN ?", vectorDocumentIDs)
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

	// 执行查询
	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Limit(limit).Offset(offset).Order("id desc").Find(&list).Error

	return list, total, err
}

func (e *FileUploadAndDownloadService) ApplyVectorScoresToResults(list []example.ExaFileUploadAndDownload, scores map[uint]float64, keyword string) []example.ExaFileUploadAndDownload {
	if scores == nil && keyword == "" {
		return list
	}

	// 为查询结果填充分数信息
	for i := range list {
		if score, exists := scores[list[i].ID]; exists {
			// 向量搜索匹配的分数（0-1之间）
			list[i].Score = score
		} else if keyword != "" && strings.Contains(strings.ToLower(list[i].Name), strings.ToLower(keyword)) {
			// 关键词精确匹配给予高分（1.0），优先于向量搜索结果
			list[i].Score = 1.0
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

// GetPresignedURL 获取文件的预签名URL（如果存储后端支持）
// 如果存储后端不支持预签名URL，则返回原始URL
func (e *FileUploadAndDownloadService) GetPresignedURL(file *example.ExaFileUploadAndDownload, expires time.Duration) string {
	if file.Key == "" {
		return file.Url
	}

	// 如果文件是公开的，直接返回原始URL，不需要预签名
	if file.PublicRead {
		return file.Url
	}

	oss := upload.NewOss()

	// 检查是否实现了预签名URL接口
	if ossWithPresigned, ok := oss.(upload.OSSWithPresignedURL); ok {
		presignedURL, err := ossWithPresigned.GetPresignedURL(file.Key, expires)
		if err != nil {
			global.GVA_LOG.Error("生成预签名URL失败，返回原始URL",
				zap.Uint("fileID", file.ID),
				zap.String("key", file.Key),
				zap.Error(err))
			return file.Url
		}
		return presignedURL
	}

	// 如果不支持预签名URL，返回原始URL
	return file.Url
}

// GetPresignedURLForFiles 批量获取文件的预签名URL
func (e *FileUploadAndDownloadService) GetPresignedURLForFiles(files []example.ExaFileUploadAndDownload, expires time.Duration) {
	for i := range files {
		files[i].Url = e.GetPresignedURL(&files[i], expires)
	}
}
