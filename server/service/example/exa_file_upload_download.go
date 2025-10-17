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

// bizMetadataToMinioMetadata 将业务元数据转换为MinIO元数据格式
func (e *FileUploadAndDownloadService) bizMetadataToMinioMetadata(bizMetadata *example.BizMetadata) map[string]string {
	if bizMetadata == nil {
		return nil
	}

	metadata := make(map[string]string)

	// 处理标签数组 - 转换为JSON字符串
	if len(bizMetadata.Tags) > 0 {
		tagsJSON, err := json.Marshal(bizMetadata.Tags)
		if err == nil {
			metadata["tags"] = string(tagsJSON)
		}
	}

	// 处理备注
	if bizMetadata.Remarks != "" {
		metadata["remarks"] = bizMetadata.Remarks
	}

	// 将整个BizMetadata结构体序列化为JSON，方便后续检索
	fullMetadataJSON, err := json.Marshal(bizMetadata)
	if err == nil {
		metadata["biz_metadata"] = string(fullMetadataJSON)
	}

	return metadata
}

func (e *FileUploadAndDownloadService) fileToMinioTags(userID uint, userName string, bizMetadata *example.BizMetadata) map[string]string {
	tags := make(map[string]string)

	maxBusinessTags := 8
	if userID > 0 {
		tags["user-id"] = fmt.Sprintf("%d", userID)
		maxBusinessTags-- // user-id占用一个标签位
	}

	if userName != "" {
		tags["username"] = userName
		maxBusinessTags-- // username占用一个标签位
	}

	if bizMetadata != nil && len(bizMetadata.Tags) > 0 {
		for i, tag := range bizMetadata.Tags {
			if i >= maxBusinessTags {
				global.GVA_LOG.Warn("MinIO标签数量超限，部分标签未设置",
					zap.Int("maxTags", maxBusinessTags),
					zap.Int("totalTags", len(bizMetadata.Tags)))
				break
			}
			// 使用tag-0, tag-1, tag-2等作为标签键
			tags[fmt.Sprintf("tag-%d", i)] = tag
		}
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
	oss := upload.NewOss()
	if err = oss.DeleteFile(fileFromDb.Key); err != nil {
		return errors.New("文件删除失败")
	}

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

func (e *FileUploadAndDownloadService) UploadFile(header *multipart.FileHeader, noSave string, classId int) (file example.ExaFileUploadAndDownload, err error) {
	return e.UploadFileWithMetadata(header, noSave, classId, 0, "", nil)
}

// UploadFileWithMetadata 上传文件并支持业务元数据
func (e *FileUploadAndDownloadService) UploadFileWithMetadata(header *multipart.FileHeader, noSave string, classId int, userID uint, userName string, bizMetadata *example.BizMetadata) (file example.ExaFileUploadAndDownload, err error) {
	// 上传前先检查文件名是否已存在
	if noSave == "0" {
		var existingFile example.ExaFileUploadAndDownload
		checkErr := global.GVA_DB.Where("name = ? AND class_id = ?", header.Filename, classId).First(&existingFile).Error
		if checkErr == nil {
			// 文件名已存在，返回错误
			global.GVA_LOG.Warn("文件名已存在，禁止上传",
				zap.String("filename", header.Filename),
				zap.Int("classId", classId),
				zap.Uint("existingFileID", existingFile.ID))
			return file, errors.New("文件名已存在")
		}
	}

	oss := upload.NewOss()

	// 转换业务元数据为MinIO元数据和标签
	minioMetadata := e.bizMetadataToMinioMetadata(bizMetadata)
	minioTags := e.fileToMinioTags(userID, userName, bizMetadata)

	var filePath, key string
	var uploadErr error

	// 检查是否为MinIO客户端，支持标签和元数据
	if minioClient, ok := oss.(*upload.Minio); ok && (minioMetadata != nil || minioTags != nil) {
		filePath, key, uploadErr = minioClient.UploadFileWithMetadataAndTags(header, minioMetadata, minioTags)
		global.GVA_LOG.Info("使用MinIO混合方案上传（元数据+标签）",
			zap.String("filename", header.Filename),
			zap.Uint("userID", userID),
			zap.String("userName", userName),
			zap.Any("metadata", minioMetadata),
			zap.Any("tags", minioTags))
	} else if ossWithMetadata, ok := oss.(interface {
		UploadFileWithMetadata(file *multipart.FileHeader, metadata map[string]string) (string, string, error)
	}); ok && minioMetadata != nil {
		// 其他支持元数据的存储后端
		filePath, key, uploadErr = ossWithMetadata.UploadFileWithMetadata(header, minioMetadata)
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
	f := example.ExaFileUploadAndDownload{
		Url:           filePath,
		Name:          header.Filename,
		ClassId:       classId,
		FileType:      s[len(s)-1],
		Key:           key,
		UserID:        userID,                       // 设置用户ID
		Username:      userName,                     // 设置用户名
		ProcessStatus: example.ProcessStatusPending, // 设置初始状态为待处理
	}

	// 如果提供了业务元数据，则设置
	if bizMetadata != nil {
		f.BizMetadata = *bizMetadata
	}
	if noSave == "0" {
		err = e.Upload(&f)
		if err != nil {
			return f, err
		}

		fileID := f.ID
		global.GVA_LOG.Info("文件上传成功，准备启动文件理解", zap.Uint64("fileID", uint64(fileID)))

		e.ProcessFileWithRetry(fileID, 3, "自动文件处理")

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
	// 设置默认值
	if topK <= 0 {
		topK = 30
	}
	if minScore < 0 {
		minScore = 0.0
	}
	if searchType < 0 || searchType > 2 {
		searchType = 2 // 默认混合检索
	}
	// 如果KnowledgeIDs为空，使用默认配置
	if len(knowledgeIDs) == 0 {
		knowledgeIDs = e.getDefaultKnowledgeIDs()
	}

	return e.searchDocumentIDsWithScores(prompt, knowledgeIDs, topK, minScore, searchType)
}

// getDefaultKnowledgeIDs 获取默认的知识库ID列表
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

// ApplyVectorScoresToResults 将向量搜索分数应用到搜索结果中
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
	// 标签过滤 - 支持多个标签，要求文件包含所有指定标签
	if len(info.Tags) > 0 {
		for _, tag := range info.Tags {
			db = db.Where("JSON_CONTAINS(biz_metadata->'$.tags', JSON_QUOTE(?))", tag)
		}
	}

	// 用户名过滤
	if info.Username != "" {
		db = db.Where("JSON_EXTRACT(biz_metadata, '$.username') = ?", info.Username)
	}
}
