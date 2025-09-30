package example

import (
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
	oss := upload.NewOss()
	filePath, key, uploadErr := oss.UploadFile(header)
	if uploadErr != nil {
		return file, uploadErr
	}
	s := strings.Split(header.Filename, ".")
	f := example.ExaFileUploadAndDownload{
		Url:           filePath,
		Name:          header.Filename,
		ClassId:       classId,
		Tag:           s[len(s)-1],
		Key:           key,
		ProcessStatus: example.ProcessStatusPending, // 设置初始状态为待处理
	}
	if noSave == "0" {
		// 检查是否已存在相同key的记录
		var existingFile example.ExaFileUploadAndDownload
		checkErr := global.GVA_DB.Where("`key` = ?", key).First(&existingFile).Error
		if checkErr == nil {
			global.GVA_LOG.Info("文件key已存在，返回现有记录",
				zap.String("key", key),
				zap.Uint("existingFileID", existingFile.ID))
			return existingFile, nil
		}

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

func (e *FileUploadAndDownloadService) GetFileRecordInfoListWithSearch(info request.ExaFileSearchRequest) (list []example.ExaFileUploadAndDownload, total int64, err error) {
	// 设置搜索默认值
	info.GetSearchDefaults()

	// 验证请求参数
	if err = info.Validate(); err != nil {
		return
	}

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

	var scores map[uint]float64
	// 向量搜索过滤
	if info.IsVectorSearch() {
		searchResult, err := e.searchDocumentIDsWithScores(
			info.Prompt,
			info.KnowledgeIDs,
			info.TopK,
			info.MinScore,
			info.SearchType,
		)
		if err != nil {
			// 向量搜索失败时记录错误但不中断流程，降级为普通搜索
			global.GVA_LOG.Warn("向量搜索失败，降级为普通搜索",
				zap.Error(err),
				zap.String("prompt", info.Prompt))
		} else if len(searchResult.DocumentIDs) == 0 {
			// 没有匹配的文档，返回空结果
			global.GVA_LOG.Info("向量搜索无匹配结果", zap.String("prompt", info.Prompt))
			return []example.ExaFileUploadAndDownload{}, 0, nil
		} else {
			// 添加向量搜索过滤条件
			db = db.Where("vectorization_document_id IN ?", searchResult.DocumentIDs)
			global.GVA_LOG.Info("应用向量搜索过滤",
				zap.String("prompt", info.Prompt),
				zap.Int("documentCount", len(searchResult.DocumentIDs)))

			// 保存分数信息
			scores = searchResult.Scores
		}
	}

	// 执行查询
	err = db.Count(&total).Error
	if err != nil {
		return
	}
	err = db.Limit(limit).Offset(offset).Order("id desc").Find(&list).Error

	// 为查询结果填充分数信息
	if scores != nil {
		for i := range list {
			if score, exists := scores[list[i].ID]; exists {
				list[i].Score = score
			}
		}

		// 按分数倒序排序
		sort.Slice(list, func(i, j int) bool {
			return list[i].Score > list[j].Score
		})
	}

	return list, total, err
}
