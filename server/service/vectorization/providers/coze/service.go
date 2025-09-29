package coze

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/pkg/vectorization"
)

// CozeService Coze向量化服务实现
type CozeService struct {
	baseURL string
	token   string
	spaceID string
	client  *http.Client
}

// NewCozeService 创建Coze服务实例
func NewCozeService(settings map[string]interface{}) (*CozeService, error) {
	baseURL, _ := settings["base_url"].(string)
	token, _ := settings["token"].(string)
	spaceID, _ := settings["space_id"].(string)

	if baseURL == "" || token == "" {
		return nil, fmt.Errorf("coze service requires base_url and token")
	}

	timeout := 30 * time.Second
	if timeoutVal, ok := settings["timeout"].(int); ok {
		timeout = time.Duration(timeoutVal) * time.Second
	}

	return &CozeService{
		baseURL: baseURL,
		token:   token,
		spaceID: spaceID,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

// CreateKnowledgeBase 创建知识库
func (c *CozeService) CreateKnowledgeBase(req *vectorization.CreateKnowledgeBaseRequest) (*vectorization.KnowledgeBase, error) {
	cozeReq := &CozeCreateKnowledgeBaseRequest{
		Name:        req.Name,
		Description: req.Description,
		SpaceID:     c.getSpaceID(req.SpaceID),
		FormatType:  req.FormatType,
		BizID:       "0",
	}

	var resp CozeCreateKnowledgeBaseResponse
	if err := c.callCozeAPI("/api/knowledge/create", cozeReq, &resp); err != nil {
		return nil, err
	}

	return &vectorization.KnowledgeBase{
		ID:          resp.DatasetID,
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		CreateTime:  time.Now().Unix(),
		Metadata:    req.Metadata,
	}, nil
}

// GetKnowledgeBase 获取知识库详情
func (c *CozeService) GetKnowledgeBase(id string) (*vectorization.KnowledgeBase, error) {
	// Coze需要通过列表API获取详情
	listReq := &vectorization.ListKnowledgeBasesRequest{
		Page: 1,
		Size: 100, // 假设不会超过100个
	}

	list, err := c.ListKnowledgeBases(listReq)
	if err != nil {
		return nil, err
	}

	for _, kb := range list.Items {
		if kb.ID == id {
			return kb, nil
		}
	}

	return nil, fmt.Errorf("knowledge base not found: %s", id)
}

// ListKnowledgeBases 获取知识库列表
func (c *CozeService) ListKnowledgeBases(req *vectorization.ListKnowledgeBasesRequest) (*vectorization.KnowledgeBaseList, error) {
	cozeReq := &CozeListKnowledgeBasesRequest{
		SpaceID:     c.spaceID,
		Page:        req.Page,
		Size:        req.Size,
		Filter:      req.Filter,
		NeedRefBots: false,
	}

	var resp CozeListKnowledgeBasesResponse
	if err := c.callCozeAPI("/api/knowledge/list", cozeReq, &resp); err != nil {
		return nil, err
	}

	items := make([]*vectorization.KnowledgeBase, len(resp.DatasetList))
	for i, dataset := range resp.DatasetList {
		items[i] = &vectorization.KnowledgeBase{
			ID:          dataset.DatasetID,
			Name:        dataset.Name,
			Description: dataset.Description,
			Status:      c.convertKnowledgeBaseStatus(dataset.Status),
			DocCount:    dataset.DocCount,
			SliceCount:  dataset.SliceCount,
			CreateTime:  dataset.CreateTime,
			UpdateTime:  dataset.UpdateTime,
		}
	}

	return &vectorization.KnowledgeBaseList{
		Items: items,
		Total: resp.Total,
	}, nil
}

// DeleteKnowledgeBase 删除知识库
func (c *CozeService) DeleteKnowledgeBase(id string) error {
	cozeReq := &CozeDeleteKnowledgeBaseRequest{
		DatasetID: id,
	}

	var resp CozeBaseResponse
	return c.callCozeAPI("/api/knowledge/delete", cozeReq, &resp)
}

// UploadDocument 上传文档
func (c *CozeService) UploadDocument(req *vectorization.UploadDocumentRequest) (*vectorization.Document, error) {
	cozeReq := &CozeUploadDocumentRequest{
		DatasetID:  req.KnowledgeBaseID,
		FormatType: vectorization.FormatTypeText,
		DocumentBases: []CozeDocumentBase{{
			Name: req.Name,
			SourceInfo: CozeSourceInfo{
				CustomContent:  req.Content,
				DocumentSource: CozeDocumentSourceCustom,
			},
		}},
		ChunkStrategy:   c.convertChunkStrategy(req.ChunkStrategy),
		ParsingStrategy: c.convertParsingStrategy(req.ParsingStrategy),
	}

	var resp CozeUploadDocumentResponse
	if err := c.callCozeAPI("/api/knowledge/document/create", cozeReq, &resp); err != nil {
		return nil, err
	}

	if len(resp.DocumentInfos) == 0 {
		return nil, fmt.Errorf("no document created")
	}

	doc := resp.DocumentInfos[0]
	return &vectorization.Document{
		ID:          doc.DocumentID,
		Name:        doc.Name,
		Status:      c.convertDocumentStatus(doc.Status),
		ContentType: req.ContentType,
		Type:        doc.Type,
		Size:        int64(doc.Size),
		CharCount:   doc.CharCount,
		SliceCount:  doc.SliceCount,
		CreateTime:  doc.CreateTime,
		Metadata:    req.Metadata,
	}, nil
}

// GetDocument 获取文档详情
func (c *CozeService) GetDocument(id string) (*vectorization.Document, error) {
	// Coze需要知道知识库ID才能获取文档，这里暂时返回错误
	// 实际使用时可能需要在数据库中存储文档与知识库的关系
	return nil, fmt.Errorf("get document by ID not supported, use ListDocuments instead")
}

// ListDocuments 获取文档列表
func (c *CozeService) ListDocuments(knowledgeBaseID string, req *vectorization.ListDocumentsRequest) (*vectorization.DocumentList, error) {
	cozeReq := &CozeListDocumentsRequest{
		DatasetID: knowledgeBaseID,
		Page:      req.Page,
		Size:      req.Size,
		Keyword:   req.Keyword,
	}

	var resp CozeListDocumentsResponse
	if err := c.callCozeAPI("/api/knowledge/document/list", cozeReq, &resp); err != nil {
		return nil, err
	}

	items := make([]*vectorization.Document, len(resp.DocumentInfos))
	for i, doc := range resp.DocumentInfos {
		items[i] = &vectorization.Document{
			ID:         doc.DocumentID,
			Name:       doc.Name,
			Status:     c.convertDocumentStatus(doc.Status),
			Type:       doc.Type,
			Size:       int64(doc.Size),
			CharCount:  doc.CharCount,
			SliceCount: doc.SliceCount,
			HitCount:   doc.HitCount,
			CreateTime: doc.CreateTime,
			UpdateTime: doc.UpdateTime,
			CreatorID:  doc.CreatorID,
		}
	}

	return &vectorization.DocumentList{
		Items: items,
		Total: resp.Total,
	}, nil
}

// DeleteDocument 删除文档
func (c *CozeService) DeleteDocument(id string) error {
	cozeReq := &CozeDeleteDocumentRequest{
		DocumentIDs: []string{id},
	}

	var resp CozeBaseResponse
	return c.callCozeAPI("/api/knowledge/document/delete", cozeReq, &resp)
}

// GetDocumentProgress 获取文档处理进度
func (c *CozeService) GetDocumentProgress(id string) (*vectorization.DocumentProgress, error) {
	cozeReq := &CozeDocumentProgressRequest{
		DocumentIDs: []string{id},
	}

	var resp CozeDocumentProgressResponse
	if err := c.callCozeAPI("/api/knowledge/document/progress/get", cozeReq, &resp); err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("document progress not found")
	}

	progress := resp.Data[0]
	return &vectorization.DocumentProgress{
		DocumentID:     progress.DocumentID,
		Progress:       progress.Progress,
		Status:         c.convertDocumentStatus(progress.Status),
		DocumentName:   progress.DocumentName,
		StatusDescript: progress.StatusDescript,
		RemainingTime:  progress.RemainingTime,
	}, nil
}

// SearchDocuments 搜索文档（暂未实现）
func (c *CozeService) SearchDocuments(req *vectorization.SearchRequest) (*vectorization.SearchResult, error) {
	// TODO: 实现基于Coze的文档搜索功能
	return nil, fmt.Errorf("search documents not implemented yet")
}

func (c *CozeService) RetrieveKnowledge(req *CozeKnowledgeRetrieveRequest) (*CozeKnowledgeRetrieveResponse, error) {
	// 设置默认值
	if req.TopK <= 0 {
		req.TopK = 10
	}
	if req.MinScore < 0 {
		req.MinScore = 0.0
	}
	if req.SearchType < 0 || req.SearchType > 2 {
		req.SearchType = 2 // 默认混合检索
	}

	// 默认启用查询重写和重排序
	req.EnableQueryRewrite = false
	req.EnableRerank = false
	req.EnableNL2SQL = false

	var resp CozeKnowledgeRetrieveResponse
	if err := c.callCozeAPI("/api/knowledge/retrieve", req, &resp); err != nil {
		return nil, fmt.Errorf("知识库检索失败: %w", err)
	}

	return &resp, nil
}

// ExtractDocumentIDs 从检索结果中提取文档ID列表
func (c *CozeService) ExtractDocumentIDs(resp *CozeKnowledgeRetrieveResponse) []string {
	if resp == nil || len(resp.Results) == 0 {
		return []string{}
	}

	// 使用map去重
	documentIDMap := make(map[int64]bool)
	for _, result := range resp.Results {
		if result.DocumentID != 0 {
			documentIDMap[result.DocumentID] = true
		}
	}

	// 转换为slice
	documentIDs := make([]string, 0, len(documentIDMap))
	for docID := range documentIDMap {
		documentIDs = append(documentIDs, fmt.Sprintf("%d", docID))
	}

	return documentIDs
}

// 辅助方法

// callCozeAPI 调用Coze API
func (c *CozeService) callCozeAPI(endpoint string, request interface{}, response interface{}) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("unmarshal response failed: %w", err)
	}

	// 检查Coze API的业务状态码
	if baseResp, ok := response.(interface{ GetCode() int }); ok {
		if code := baseResp.GetCode(); code != 0 {
			return fmt.Errorf("coze API error: code=%d", code)
		}
	}

	return nil
}

// getSpaceID 获取SpaceID，优先使用请求中的，否则使用配置的
func (c *CozeService) getSpaceID(requestSpaceID string) string {
	if requestSpaceID != "" {
		return requestSpaceID
	}
	return c.spaceID
}

// convertKnowledgeBaseStatus 转换知识库状态
func (c *CozeService) convertKnowledgeBaseStatus(status int) string {
	switch status {
	case 1:
		return "active"
	case 0:
		return "processing"
	default:
		return "unknown"
	}
}

// convertDocumentStatus 转换文档状态
func (c *CozeService) convertDocumentStatus(status int) string {
	switch status {
	case CozeDocumentStatusProcessing:
		return vectorization.DocumentStatusProcessing
	case CozeDocumentStatusEnable:
		return vectorization.DocumentStatusEnable
	case CozeDocumentStatusDisable:
		return vectorization.DocumentStatusDisable
	case CozeDocumentStatusDeleted:
		return vectorization.DocumentStatusDeleted
	case CozeDocumentStatusResegment:
		return vectorization.DocumentStatusResegment
	case CozeDocumentStatusRefreshing:
		return vectorization.DocumentStatusRefreshing
	case CozeDocumentStatusFailed:
		return vectorization.DocumentStatusFailed
	default:
		return vectorization.DocumentStatusFailed
	}
}

// convertChunkStrategy 转换分块策略
func (c *CozeService) convertChunkStrategy(strategy *vectorization.ChunkStrategy) *CozeChunkStrategy {
	if strategy == nil {
		return &CozeChunkStrategy{
			Separator:         ".",
			MaxTokens:         1000,
			RemoveExtraSpaces: true,
			ChunkType:         vectorization.ChunkTypeDefault,
		}
	}

	return &CozeChunkStrategy{
		Separator:         strategy.Separator,
		MaxTokens:         strategy.MaxTokens,
		RemoveExtraSpaces: strategy.RemoveExtraSpaces,
		ChunkType:         strategy.ChunkType,
		Overlap:           strategy.Overlap,
	}
}

// convertParsingStrategy 转换解析策略
func (c *CozeService) convertParsingStrategy(strategy *vectorization.ParsingStrategy) *CozeParsingStrategy {
	if strategy == nil {
		return &CozeParsingStrategy{
			ParsingType:     1,
			ImageExtraction: true,
			TableExtraction: true,
			ImageOCR:        false,
		}
	}

	return &CozeParsingStrategy{
		ParsingType:     strategy.ParsingType,
		ImageExtraction: strategy.ImageExtraction,
		TableExtraction: strategy.TableExtraction,
		ImageOCR:        strategy.ImageOCR,
	}
}

// 为了支持业务状态码检查，给响应结构体添加方法
func (r *CozeCreateKnowledgeBaseResponse) GetCode() int { return r.Code }
func (r *CozeListKnowledgeBasesResponse) GetCode() int  { return r.Code }
func (r *CozeUploadDocumentResponse) GetCode() int      { return r.Code }
func (r *CozeListDocumentsResponse) GetCode() int       { return r.Code }
func (r *CozeDocumentProgressResponse) GetCode() int    { return r.Code }
func (r *CozeKnowledgeRetrieveResponse) GetCode() int   { return r.Code }
func (r *CozeBaseResponse) GetCode() int                { return r.Code }
