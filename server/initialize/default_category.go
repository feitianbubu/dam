package initialize

import (
	"context"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/pkg/vectorization"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
	vectorizationService "github.com/flipped-aurora/gin-vue-admin/server/service/vectorization"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const initOrderDefaultCategory = system.InitOrderInternal

type initDefaultCategory struct{}

// auto run
func init() {
	system.RegisterInit(initOrderDefaultCategory, &initDefaultCategory{})
}

func (i *initDefaultCategory) InitializerName() string {
	return "default_attachment_category"
}

func (i *initDefaultCategory) MigrateTable(ctx context.Context) (context.Context, error) {
	return ctx, nil // 表已经在 ensure_tables 中创建
}

func (i *initDefaultCategory) TableCreated(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	return db.Migrator().HasTable(&example.ExaAttachmentCategory{})
}

func (i *initDefaultCategory) InitializeData(ctx context.Context) (next context.Context, err error) {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return ctx, system.ErrMissingDBContext
	}

	// 检查是否已存在默认分类
	var count int64
	if err := db.Model(&example.ExaAttachmentCategory{}).Where("id = ?", 1).Count(&count).Error; err != nil {
		return ctx, fmt.Errorf("检查默认分类失败: %w", err)
	}

	if count > 0 {
		global.GVA_LOG.Info("默认分类已存在，跳过初始化")
		return ctx, nil
	}

	// 创建知识库（如果向量化启用）
	var knowledgeID string
	if global.GVA_CONFIG.Vectorization.Enable {
		knowledgeID, err = i.createDefaultKnowledgeBase()
		if err != nil {
			// 向量化失败不应阻止系统启动，记录警告即可
			global.GVA_LOG.Warn("创建默认知识库失败，将创建无知识库的默认分类", zap.Error(err))
			knowledgeID = ""
		}
	}

	// 创建默认分类
	defaultCategory := &example.ExaAttachmentCategory{
		Name:        "未分类",
		Pid:         0,
		KnowledgeID: knowledgeID,
	}
	// 使用原始 SQL 强制设置 ID 为 1
	defaultCategory.ID = 1

	if err := db.Create(defaultCategory).Error; err != nil {
		// 如果创建失败且已创建知识库，尝试删除知识库
		if knowledgeID != "" {
			_ = i.deleteKnowledgeBase(knowledgeID)
		}
		return ctx, fmt.Errorf("创建默认分类失败: %w", err)
	}

	global.GVA_LOG.Info("默认分类初始化成功",
		zap.Uint("categoryID", defaultCategory.ID),
		zap.String("knowledgeID", knowledgeID))

	return ctx, nil
}

func (i *initDefaultCategory) DataInserted(ctx context.Context) bool {
	db, ok := ctx.Value("db").(*gorm.DB)
	if !ok {
		return false
	}
	var count int64
	if err := db.Model(&example.ExaAttachmentCategory{}).Where("id = ?", 1).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// createDefaultKnowledgeBase 创建默认知识库
func (i *initDefaultCategory) createDefaultKnowledgeBase() (string, error) {
	vectorService, err := vectorizationService.NewVectorizationServiceFromGlobalConfig()
	if err != nil {
		return "", fmt.Errorf("初始化向量化服务失败: %w", err)
	}

	// 生成知识库名称
	bucketName := global.GVA_CONFIG.Minio.BucketName
	if bucketName == "" {
		bucketName = "dam_default"
	}
	kbName := fmt.Sprintf("%s_class_default_vector", bucketName)

	// 创建知识库
	req := &vectorization.CreateKnowledgeBaseRequest{
		Name:        kbName,
		Description: "Dam系统默认分类的知识库，用于存储未指定分类的文件",
		FormatType:  vectorization.FormatTypeText,
	}

	kb, err := vectorService.CreateKnowledgeBase(req)
	if err != nil {
		return "", fmt.Errorf("创建知识库失败: %w", err)
	}

	global.GVA_LOG.Info("默认知识库创建成功",
		zap.String("knowledgeBaseID", kb.ID),
		zap.String("knowledgeBaseName", kb.Name))

	return kb.ID, nil
}

// deleteKnowledgeBase 删除知识库（失败回滚时使用）
func (i *initDefaultCategory) deleteKnowledgeBase(knowledgeID string) error {
	vectorService, err := vectorizationService.NewVectorizationServiceFromGlobalConfig()
	if err != nil {
		global.GVA_LOG.Warn("初始化向量化服务失败，无法删除知识库",
			zap.String("knowledgeID", knowledgeID),
			zap.Error(err))
		return err
	}

	if err := vectorService.DeleteKnowledgeBase(knowledgeID); err != nil {
		global.GVA_LOG.Warn("删除知识库失败",
			zap.String("knowledgeID", knowledgeID),
			zap.Error(err))
		return err
	}

	return nil
}
