package example

import (
	"errors"
	"fmt"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/pkg/vectorization"
	vectorizationService "github.com/flipped-aurora/gin-vue-admin/server/service/vectorization"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AttachmentCategoryService struct{}

// AddCategory 创建/更新的分类
func (a *AttachmentCategoryService) AddCategory(req *example.ExaAttachmentCategory) (err error) {
	// 检查是否已存在相同名称的分类
	if (!errors.Is(global.GVA_DB.Take(&example.ExaAttachmentCategory{}, "name = ? and pid = ?", req.Name, req.Pid).Error, gorm.ErrRecordNotFound)) {
		return errors.New("分类名称已存在")
	}

	// 如果是更新操作
	if req.ID > 0 {
		if err = global.GVA_DB.Model(&example.ExaAttachmentCategory{}).Where("id = ?", req.ID).Updates(&example.ExaAttachmentCategory{
			Name: req.Name,
			Pid:  req.Pid,
		}).Error; err != nil {
			return err
		}
		return nil
	}

	// 创建操作：需要先创建知识库
	var knowledgeID string
	if global.GVA_CONFIG.Vectorization.Enable {
		knowledgeID, err = a.createKnowledgeBaseForCategory(req.Name)
		if err != nil {
			return fmt.Errorf("创建知识库失败: %w", err)
		}
	}

	// 创建分类记录
	category := &example.ExaAttachmentCategory{
		Name:        req.Name,
		Pid:         req.Pid,
		KnowledgeID: knowledgeID,
	}

	if err = global.GVA_DB.Create(category).Error; err != nil {
		// 创建失败，尝试删除已创建的知识库
		if knowledgeID != "" {
			_ = a.deleteKnowledgeBase(knowledgeID)
		}
		return err
	}

	return nil
}

// DeleteCategory 删除分类
func (a *AttachmentCategoryService) DeleteCategory(id *int) error {
	var childCount int64
	global.GVA_DB.Model(&example.ExaAttachmentCategory{}).Where("pid = ?", id).Count(&childCount)
	if childCount > 0 {
		return errors.New("请先删除子级")
	}
	return global.GVA_DB.Where("id = ?", id).Unscoped().Delete(&example.ExaAttachmentCategory{}).Error
}

// GetCategoryList 分类列表
func (a *AttachmentCategoryService) GetCategoryList() (res []*example.ExaAttachmentCategory, err error) {
	var fileLists []example.ExaAttachmentCategory
	err = global.GVA_DB.Model(&example.ExaAttachmentCategory{}).Find(&fileLists).Error
	if err != nil {
		return res, err
	}
	return a.getChildrenList(fileLists, 0), nil
}

// getChildrenList 子类
func (a *AttachmentCategoryService) getChildrenList(categories []example.ExaAttachmentCategory, parentID uint) []*example.ExaAttachmentCategory {
	var tree []*example.ExaAttachmentCategory
	for _, category := range categories {
		if category.Pid == parentID {
			category.Children = a.getChildrenList(categories, category.ID)
			tree = append(tree, &category)
		}
	}
	return tree
}

// createKnowledgeBaseForCategory 为分类创建知识库
func (a *AttachmentCategoryService) createKnowledgeBaseForCategory(categoryName string) (string, error) {
	// 创建向量化服务实例
	vectorService, err := vectorizationService.NewVectorizationServiceFromGlobalConfig()
	if err != nil {
		return "", fmt.Errorf("初始化向量化服务失败: %w", err)
	}

	// 生成知识库名称
	bucketName := global.GVA_CONFIG.Minio.BucketName
	if bucketName == "" {
		bucketName = "dam_default"
	}
	kbName := fmt.Sprintf("%s_class_%s_vector", bucketName, categoryName)

	// 创建知识库
	req := &vectorization.CreateKnowledgeBaseRequest{
		Name:        kbName,
		Description: fmt.Sprintf("Dam系统为分类'%s'自动创建的知识库", categoryName),
		FormatType:  vectorization.FormatTypeText,
	}

	kb, err := vectorService.CreateKnowledgeBase(req)
	if err != nil {
		return "", fmt.Errorf("创建知识库失败: %w", err)
	}

	global.GVA_LOG.Info("为分类创建知识库成功",
		zap.String("categoryName", categoryName),
		zap.String("knowledgeBaseID", kb.ID),
		zap.String("knowledgeBaseName", kb.Name))

	return kb.ID, nil
}

// deleteKnowledgeBase 删除知识库（失败回滚时使用）
func (a *AttachmentCategoryService) deleteKnowledgeBase(knowledgeID string) error {
	vectorService, err := vectorizationService.NewVectorizationServiceFromGlobalConfig()
	if err != nil {
		global.GVA_LOG.Warn("初始化向量化服务失败，无法删除知识库", zap.String("knowledgeID", knowledgeID), zap.Error(err))
		return err
	}

	if err := vectorService.DeleteKnowledgeBase(knowledgeID); err != nil {
		global.GVA_LOG.Warn("删除知识库失败", zap.String("knowledgeID", knowledgeID), zap.Error(err))
		return err
	}

	return nil
}
